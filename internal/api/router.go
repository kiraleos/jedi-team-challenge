package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(apiHandler *APIHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)       // Basic request logging
	r.Use(middleware.Recoverer)    // Recover from panics
	r.Use(middleware.StripSlashes) // Ensure consistent path handling

	// All API routes will be under /api
	r.Route("/api", func(r chi.Router) {
		// Public routes
		r.Post("/login", apiHandler.LoginHandler)
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})

		// User-authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(apiHandler.JWTAuthMiddleware)

			// Chat routes
			r.Post("/chats", apiHandler.CreateChatHandler)
			r.Get("/chats", apiHandler.ListChatsHandler)
			r.Get("/chats/{chatID}", apiHandler.GetChatDetailsHandler)
			r.Post("/chats/{chatID}/messages", apiHandler.PostMessageHandler)

			// Message feedback route
			r.Post("/messages/{messageID}/feedback", apiHandler.MessageFeedbackHandler)
		})
	})

	return r
}
