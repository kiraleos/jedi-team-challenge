package api

import (
	"encoding/json"
	"log"
	"net/http"

	"gwi.com/jedi-team-challenge/internal/core"
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

	creds := core.SignupCredentials{
		UserID:   req.UserID,
		Password: req.Password,
	}

	user, err := h.authService.Signup(creds)
	if err != nil {
		log.Printf("Error signing up user %s: %v", req.UserID, err)
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

	creds := core.LoginCredentials{
		UserID:   req.UserID,
		Password: req.Password,
	}

	token, err := h.authService.Login(creds)
	if err != nil {
		log.Printf("Error logging in user %s: %v", req.UserID, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
