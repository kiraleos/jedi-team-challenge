package core

import (
	"fmt"
	"log"
	"strings"

	"gwi.com/jedi-team-challenge/internal/store"
)

type ChatService struct {
	dbStore    *store.SQLiteStore
	ragService *RAGService
	llmService *LLMService // For title generation
}

func NewChatService(db *store.SQLiteStore, rag *RAGService, llm *LLMService) *ChatService {
	return &ChatService{
		dbStore:    db,
		ragService: rag,
		llmService: llm,
	}
}

// GetOrCreateUser ensures a user exists and returns their internal ID.
func (s *ChatService) GetOrCreateUser(externalUserID string) (*store.User, error) {
	return s.dbStore.GetOrCreateUser(externalUserID)
}

func (s *ChatService) CreateChat(userID int64, firstMessageContent *string) (*store.Chat, []store.Message, error) {
	// This should ideally be wrapped in a transaction

	chat, err := s.dbStore.CreateChat(userID, nil) // Title will be generated later
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create chat in DB: %w", err)
	}

	var messages []store.Message

	if firstMessageContent != nil && *firstMessageContent != "" {
		userMsg := store.Message{
			ChatID:  chat.ID,
			Sender:  "user",
			Content: *firstMessageContent,
		}
		if err := s.dbStore.CreateMessage(&userMsg); err != nil {
			// Potentially rollback chat creation or log warning
			log.Printf("Failed to store first user message for new chat %s: %v", chat.ID, err)
			// Continue, but the chat will be empty initially
		} else {
			messages = append(messages, userMsg)

			// Generate model response for the first message
			modelContent, err := s.ragService.GenerateResponse(chat.ID, userID, userMsg.Content)
			if err != nil {
				// Log error, but still return the chat and user message
				log.Printf("Failed to generate initial model response for chat %s: %v", chat.ID, err)
				modelContent = "I encountered an issue trying to respond. Please try again."
			}

			modelMsg := store.Message{
				ChatID:  chat.ID,
				Sender:  "model",
				Content: modelContent,
			}
			if err := s.dbStore.CreateMessage(&modelMsg); err != nil {
				log.Printf("Failed to store initial model message for new chat %s: %v", chat.ID, err)
			} else {
				messages = append(messages, modelMsg)
			}

			// Auto-generate title after first exchange
			go s.generateAndSaveChatTitle(chat.ID, userID, userMsg.Content)
		}
	}

	return chat, messages, nil
}

func (s *ChatService) GetChats(userID int64) ([]store.Chat, error) {
	return s.dbStore.GetChatsByUserID(userID)
}

func (s *ChatService) GetChatDetails(chatID string, userID int64) (*store.Chat, []store.Message, error) {
	chat, err := s.dbStore.GetChatByID(chatID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chat: %w", err)
	}
	if chat == nil {
		return nil, nil, nil // Not found
	}

	messages, err := s.dbStore.GetMessagesByChatID(chatID, 100, 0) // Get up to 100 messages
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get messages for chat: %w", err)
	}
	return chat, messages, nil
}

func (s *ChatService) PostMessage(chatID string, userID int64, userContent string) (*store.Message, error) {
	// This should ideally be wrapped in a transaction

	// Verify chat exists and belongs to user
	chat, err := s.dbStore.GetChatByID(chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify chat: %w", err)
	}
	if chat == nil {
		return nil, fmt.Errorf("chat not found")
	}

	// Store user message
	userMsg := store.Message{
		ChatID:  chatID,
		Sender:  "user",
		Content: userContent,
	}
	if err := s.dbStore.CreateMessage(&userMsg); err != nil {
		return nil, fmt.Errorf("failed to store user message: %w", err)
	}

	// Generate model response using RAG service
	modelContent, err := s.ragService.GenerateResponse(chatID, userID, userContent)
	if err != nil {
		// Log error, maybe return a canned error message to user
		log.Printf("Error generating model response for chat %s: %v", chatID, err)
		modelContent = "I'm sorry, I encountered an error while processing your request."
	}

	// Store model message
	modelMessage := store.Message{
		ChatID:  chatID,
		Sender:  "model",
		Content: modelContent,
	}
	if err := s.dbStore.CreateMessage(&modelMessage); err != nil {
		return nil, fmt.Errorf("failed to store model message: %w", err)
	}

	// If chat doesn't have a title yet (e.g. created without a first message, or title generation failed)
	// attempt to generate it now.
	if chat.Title == nil || *chat.Title == "" {
		// Check if there are at least a user and a model message now.
		messages, _ := s.dbStore.GetLastNMessagesByChatID(chatID, 2)
		if len(messages) >= 1 { // If only one message, use it. Better with 2 for context.
			firstUserMessageContent := ""
			for _, m := range messages {
				if m.Sender == "user" {
					firstUserMessageContent = m.Content
					break
				}
			}
			if firstUserMessageContent != "" {
				go s.generateAndSaveChatTitle(chatID, userID, firstUserMessageContent)
			}
		}
	}

	return &modelMessage, nil
}

func (s *ChatService) generateAndSaveChatTitle(chatID string, userID int64, basisContent string) {
	log.Printf("Attempting to generate title for chat %s", chatID)
	title, err := s.llmService.GenerateTitleForChat(basisContent)
	if err != nil {
		log.Printf("Failed to generate title for chat %s: %v", chatID, err)
		return
	}
	title = strings.Trim(title, "\"'\n\r\t .")

	err = s.dbStore.UpdateChatTitle(chatID, userID, title)
	if err != nil {
		log.Printf("Failed to save generated title '%s' for chat %s: %v", title, chatID, err)
	} else {
		log.Printf("Successfully generated and saved title '%s' for chat %s", title, chatID)
	}
}

func (s *ChatService) SetMessageFeedback(messageID string, userID int64, negative bool) error {
	// Should verify that the message belongs to the user's chat
	return s.dbStore.UpdateMessageFeedback(messageID, negative)
}
