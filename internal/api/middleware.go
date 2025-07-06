package api

import (
	"context"
	"log"
	"net/http"
	"strings"

	"gwi.com/jedi-team-challenge/internal/auth"
)

type contextKey string

const (
	userIDKey         contextKey = "userID"
	externalUserIDKey contextKey = "externalUserID"
)

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

		ctx := context.WithValue(r.Context(), userIDKey, user.ID)
		ctx = context.WithValue(ctx, externalUserIDKey, user.ExternalUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
