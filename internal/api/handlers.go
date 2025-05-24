package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gwi.com/jedi-team-challenge/internal/core"
	"gwi.com/jedi-team-challenge/internal/store"
)

type APIHandler struct {
	chatService *core.ChatService
}

func NewAPIHandler(cs *core.ChatService) *APIHandler {
	return &APIHandler{chatService: cs}
}

// Middleware to extract and validate X-User-ID
func (h *APIHandler) UserAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalUserID := r.Header.Get("X-User-ID")
		if externalUserID == "" {
			http.Error(w, "X-User-ID header is required", http.StatusUnauthorized)
			return
		}
		// Get or create user, attach internal user ID to context
		user, err := h.chatService.GetOrCreateUser(externalUserID)
		if err != nil {
			log.Printf("Error in UserAuthMiddleware for X-User-ID %s: %v", externalUserID, err)
			http.Error(w, "Failed to process user identity", http.StatusInternalServerError)
			return
		}
		ctx := context.WithValue(r.Context(), "userID", user.ID)
		ctx = context.WithValue(ctx, "externalUserID", user.ExternalUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
