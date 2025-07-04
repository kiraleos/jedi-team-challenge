package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"gwi.com/jedi-team-challenge/internal/auth"
	"gwi.com/jedi-team-challenge/internal/core"
	"gwi.com/jedi-team-challenge/internal/store"
)

type APIHandler struct {
	chatService *core.ChatService
}

func NewAPIHandler(cs *core.ChatService) *APIHandler {
	return &APIHandler{chatService: cs}
}

func (h *APIHandler) JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		externalUserID, err := auth.ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		user, err := h.chatService.GetUserByExternalID(externalUserID)
		if err != nil {
			log.Printf("Error in JWTAuthMiddleware for user %s: %v", externalUserID, err)
			http.Error(w, "Failed to process user identity", http.StatusInternalServerError)
			return
		}

		if user == nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", user.ID)
		ctx = context.WithValue(ctx, "externalUserID", user.ExternalUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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

type CreateChatRequest struct {
	FirstMessage *string `json:"first_message,omitempty"`
}

type CreateChatResponse struct {
	*store.Chat
	Messages []store.Message `json:"messages,omitempty"`
}

func (h *APIHandler) CreateChatHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int64)

	var req CreateChatRequest
	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	chat, messages, err := h.chatService.CreateChat(userID, req.FirstMessage)
	if err != nil {
		log.Printf("Error creating chat for user %d: %v", userID, err)
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	resp := CreateChatResponse{
		Chat:     chat,
		Messages: messages,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *APIHandler) ListChatsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int64)

	chats, err := h.chatService.GetChats(userID)
	if err != nil {
		log.Printf("Error listing chats for user %d: %v", userID, err)
		http.Error(w, "Failed to list chats", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(chats)
}

type GetChatDetailsResponse struct {
	*store.Chat
	Messages []store.Message `json:"messages"`
}

func (h *APIHandler) GetChatDetailsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int64)
	chatID := chi.URLParam(r, "chatID")

	chat, messages, err := h.chatService.GetChatDetails(chatID, userID)
	if err != nil {
		log.Printf("Error getting chat details for user %d, chat %s: %v", userID, chatID, err)
		http.Error(w, "Failed to get chat details", http.StatusInternalServerError)
		return
	}
	if chat == nil {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	resp := GetChatDetailsResponse{
		Chat:     chat,
		Messages: messages,
	}
	json.NewEncoder(w).Encode(resp)
}

type PostMessageRequest struct {
	Content string `json:"content"`
}

func (h *APIHandler) PostMessageHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int64)
	chatID := chi.URLParam(r, "chatID")

	var req PostMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		http.Error(w, "Message content cannot be empty", http.StatusBadRequest)
		return
	}

	modelMessage, err := h.chatService.PostMessage(chatID, userID, req.Content)
	if err != nil {
		if err.Error() == "chat not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			log.Printf("Error posting message for user %d, chat %s: %v", userID, chatID, err)
			http.Error(w, "Failed to post message", http.StatusInternalServerError)
		}
		return
	}
	json.NewEncoder(w).Encode(modelMessage)
}

type FeedbackRequest struct {
	Negative bool `json:"negative"`
}

func (h *APIHandler) MessageFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int64)
	messageID := chi.URLParam(r, "messageID")

	var req FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	err := h.chatService.SetMessageFeedback(messageID, userID, req.Negative)
	if err != nil {
		if err.Error() == "message not found for feedback" {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			log.Printf("Error setting feedback for message %s by user %d: %v", messageID, userID, err)
			http.Error(w, "Failed to set feedback", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
