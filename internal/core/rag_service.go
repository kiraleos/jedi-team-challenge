package core

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"gwi.com/jedi-team-challenge/internal/store"
	"gwi.com/jedi-team-challenge/internal/utils"
)

const (
	NumRelevantChunks   = 3   // Number of chunks to retrieve for context
	SimilarityThreshold = 0.7 // Minimum similarity score to consider a chunk relevant
)

type RAGService struct {
	dbStore    *store.SQLiteStore
	llmService *LLMService
	dataChunks []store.DataChunk // In-memory cache of data chunks and their embeddings
}

func NewRAGService(db *store.SQLiteStore, llm *LLMService) (*RAGService, error) {
	chunks, err := db.GetAllDataChunks()
	if err != nil {
		return nil, fmt.Errorf("failed to load data chunks for RAG service: %w", err)
	}
	if len(chunks) == 0 {
		log.Println("Warning: RAGService initialized with no data chunks. Ensure data has been ingested with the current embedding model.")
	} else {
		log.Printf("RAGService initialized with %d data chunks.", len(chunks))
	}

	return &RAGService{
		dbStore:    db,
		llmService: llm,
		dataChunks: chunks,
	}, nil
}

type ScoredChunk struct {
	Chunk      store.DataChunk
	Similarity float32
}

func (s *RAGService) GetRelevantContext(query string) (string, error) {
	if len(s.dataChunks) == 0 {
		log.Println("No data chunks available for RAG context retrieval.")
		return "", nil // No context if no data
	}

	queryEmbedding, err := s.llmService.GetEmbedding(query)
	if err != nil {
		return "", fmt.Errorf("failed to get query embedding: %w", err)
	}

	scoredChunks := make([]ScoredChunk, 0, len(s.dataChunks))
	for _, chunk := range s.dataChunks {
		if len(chunk.Embedding) == 0 {
			log.Printf("Skipping chunk ID %d due to missing embedding.", chunk.ID)
			continue
		}
		similarity, err := utils.CosineSimilarity(queryEmbedding, chunk.Embedding)
		if err != nil {
			log.Printf("Error calculating similarity for chunk %d with query: %v. Skipping.", chunk.ID, err)
			continue // Skip this chunk
		}

		if similarity >= SimilarityThreshold { // Only consider similar chunks over the threshold
			scoredChunks = append(scoredChunks, ScoredChunk{Chunk: chunk, Similarity: similarity})
		}
	}

	// Sort by similarity in descending order
	sort.Slice(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].Similarity > scoredChunks[j].Similarity
	})

	var contextBuilder strings.Builder
	retrievedCount := 0
	for i := 0; i < len(scoredChunks) && retrievedCount < NumRelevantChunks; i++ {
		contextBuilder.WriteString(scoredChunks[i].Chunk.Content)
		contextBuilder.WriteString("\n\n") // Separate chunks clearly
		retrievedCount++
	}

	if retrievedCount == 0 {
		log.Printf("No relevant chunks found for query (Similarity threshold: %.2f): %s", SimilarityThreshold, query)
		return "", nil // No relevant context found meeting the threshold
	}

	log.Printf("Retrieved %d relevant chunks for query.", retrievedCount)
	return strings.TrimSpace(contextBuilder.String()), nil
}

func (s *RAGService) GenerateResponse(chatID string, userID int64, userQuery string) (string, error) {
	// 1. Retrieve chat history (last few messages)
	chatHistoryMsgs, err := s.dbStore.GetLastNMessagesByChatID(chatID, 5) // Get last 5 messages (in order to avoid too long history)
	if err != nil {
		log.Printf("Error getting chat history for chat %s: %v. Proceeding without history.", chatID, err)
		chatHistoryMsgs = []store.Message{}
	}

	// 2. Get relevant context from RAG
	relevantContext, err := s.GetRelevantContext(userQuery)
	if err != nil {
		// Don't fail the whole request if context retrieval fails, just log and proceed without context.
		// The LLM might still be able to answer based on history or general knowledge if allowed.
		log.Printf("Failed to get relevant context, proceeding without it: %v", err)
		relevantContext = "" // Ensure it's an empty string
	}

	// 3. Construct prompt for Gemini
	// The SystemInstruction is set on the model in LLMService.
	var geminiChatHistory []*genai.Content

	// Add chat history to prompt
	for _, msg := range chatHistoryMsgs {
		geminiChatHistory = append(geminiChatHistory, &genai.Content{
			Role:  msg.Sender,
			Parts: []genai.Part{genai.Text(msg.Content)},
		})
	}

	// Add RAG context and current user query as the last "user" turn
	finalUserContent := ""
	if relevantContext != "" {
		finalUserContent = fmt.Sprintf("Based on our previous conversation and the following potentially relevant context from GWI market research data:\n\n--- CONTEXT START ---\n%s\n--- CONTEXT END ---\n\nNow, please answer my question: %s", relevantContext, userQuery)
	} else {
		finalUserContent = fmt.Sprintf("Based on our previous conversation (if any), and noting that I couldn't find specific GWI documents for your current question, please answer: %s", userQuery)
	}

	geminiChatHistory = append(geminiChatHistory, &genai.Content{
		Role:  "user",
		Parts: []genai.Part{genai.Text(finalUserContent)},
	})

	// 4. Get response from LLM
	modelResponse, err := s.llmService.GetChatCompletion(geminiChatHistory)
	if err != nil {
		return "", fmt.Errorf("failed to get LLM completion: %w", err)
	}

	return modelResponse, nil
}
