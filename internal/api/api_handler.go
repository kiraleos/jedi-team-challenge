package api

import (
	"gwi.com/jedi-team-challenge/internal/core"
)

type APIHandler struct {
	chatService *core.ChatService
	authService *core.AuthService
}

func NewAPIHandler(cs *core.ChatService, as *core.AuthService) *APIHandler {
	return &APIHandler{
		chatService: cs,
		authService: as,
	}
}
