package core

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"gwi.com/jedi-team-challenge/internal/config"
)

const (
	defaultChatModelName      = "gemini-1.5-flash-latest"
	defaultEmbeddingModelName = "text-embedding-004"
	defaultTitleModelName     = "gemini-1.5-flash-latest"

	chatSystemInstruction = "You are a helpful GWI assistant. Answer questions based on the provided market research data. " +
		"If the answer is not found in the provided context, clearly state that you don't have the information. " +
		"Keep your answers concise and directly related to the user's question and provided context. " +
		"Do not make up information. If the context is insufficient, say so."

	titleSystemInstruction = "You are a helpful assistant that generates concise titles for chat conversations. " +
		"The title should be 3-5 words maximum. Just return the title itself, nothing else."
)

type LLMService struct {
	client *genai.Client
}

func NewLLMService() *LLMService {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(config.AppConfig.GeminiAPIKey))
	if err != nil {
		log.Fatalf("Failed to create GenAI client: %v", err)
	}

	return &LLMService{
		client: client,
	}
}

func (s *LLMService) Close() {
	if s.client != nil {
		if err := s.client.Close(); err != nil {
			log.Printf("Error closing GenAI client: %v", err)
		} else {
			log.Println("GenAI client closed.")
		}
	}
}

func (s *LLMService) GetEmbedding(text string) ([]float32, error) {
	ctx := context.Background()
	em := s.client.EmbeddingModel(defaultEmbeddingModelName)
	res, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("gemini embedding request failed: %w", err)
	}

	if res.Embedding == nil || len(res.Embedding.Values) == 0 {
		return nil, fmt.Errorf("no embedding data received from gemini")
	}
	return res.Embedding.Values, nil
}

func (s *LLMService) GetChatCompletion(promptHistory []*genai.Content) (string, error) {
	ctx := context.Background()
	model := s.client.GenerativeModel(defaultChatModelName)

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(chatSystemInstruction)},
	}

	chatSession := model.StartChat()
	chatSession.History = promptHistory

	if len(promptHistory) == 0 {
		return "", fmt.Errorf("prompt history is empty for chat completion")
	}

	lastUserMessage := promptHistory[len(promptHistory)-1]
	if lastUserMessage.Role != "user" {
		// This should ideally not happen if RAG service constructs correctly
		return "", fmt.Errorf("last message in history is not from 'user', cannot proceed with chat completion")
	}

	// Send the parts of the last user message.
	resp, err := chatSession.SendMessage(ctx, lastUserMessage.Parts...)
	if err != nil {
		return "", fmt.Errorf("gemini chat SendMessage failed: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Println("Gemini response was empty or had no valid candidates/parts.")
		return "I'm sorry, I couldn't generate a response at this time. Please try again.", nil
	}

	var responseText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			responseText.WriteString(string(txt))
		} else {
			log.Printf("Gemini response part was not text: %T", part)
		}
	}

	if responseText.Len() == 0 {
		log.Println("Gemini response part was not text or was empty after processing.")
		return "I received an empty or non-text response, please try rephrasing your question.", nil
	}

	return responseText.String(), nil
}

func (s *LLMService) GenerateTitleForChat(chatSummary string) (string, error) {
	ctx := context.Background()
	model := s.client.GenerativeModel(defaultTitleModelName)

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(titleSystemInstruction)},
	}

	temp := float32(0.3)
	maxTokens := int32(20)

	model.GenerationConfig = genai.GenerationConfig{
		MaxOutputTokens: &maxTokens,
		Temperature:     &temp,
	}

	userPromptForTitle := fmt.Sprintf("Generate a very concise title (3-5 words maximum) for a conversation that starts with or is about: \"%s\".", chatSummary)

	resp, err := model.GenerateContent(ctx, genai.Text(userPromptForTitle))
	if err != nil {
		return "", fmt.Errorf("gemini title generation request failed: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "Chat", fmt.Errorf("LLM did not generate a title (empty response)")
	}

	var titleText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			titleText.WriteString(string(txt))
		}
	}

	if titleText.Len() == 0 {
		return "Chat", fmt.Errorf("LLM generated an empty title string")
	}

	return strings.Trim(titleText.String(), "\"'\n\r\t ."), nil
}
