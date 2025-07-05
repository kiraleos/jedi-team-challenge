package api

import (
	"encoding/json"
	"log"
	"net/http"

	"gwi.com/jedi-team-challenge/internal/auth"
)

type SignupRequest struct {
	UserID   string `json:"user_id"`
	Password string `json:"password"`
}

func (h *APIHandler) SignupHandler(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.Password == "" {
		http.Error(w, "User ID and password are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	user, err := h.chatService.CreateUser(req.UserID, hashedPassword)
	if err != nil {
		log.Printf("Error creating user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

type LoginRequest struct {
	UserID   string `json:"user_id"`
	Password string `json:"password"`
}

func (h *APIHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.Password == "" {
		http.Error(w, "User ID and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.chatService.GetUserByExternalID(req.UserID)
	if err != nil {
		log.Printf("Error getting user %s: %v", req.UserID, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if user == nil || !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateJWT(req.UserID)
	if err != nil {
		log.Printf("Error generating JWT for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
