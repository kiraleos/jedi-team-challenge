package api

import (
	"gwi.com/jedi-team-challenge/internal/core"
)

type APIHandler struct {
	chatService *core.ChatService
}

func NewAPIHandler(cs *core.ChatService) *APIHandler {
	return &APIHandler{chatService: cs}
}