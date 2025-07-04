package store

import "time"

type User struct {
	ID             int64     `json:"id"`
	ExternalUserID string    `json:"external_user_id"`
	PasswordHash   string    `json:"-"` // Do not expose this in JSON responses
	CreatedAt      time.Time `json:"created_at"`
}

type Chat struct {
	ID        string    `json:"id"` // Using UUID for external ID
	UserID    int64     `json:"user_id"`
	Title     *string   `json:"title"` // Nullable
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID               string    `json:"id"` // Using UUID for external ID
	ChatID           string    `json:"chat_id"`
	Sender           string    `json:"sender"` // "user" or "model"
	Content          string    `json:"content"`
	Timestamp        time.Time `json:"timestamp"`
	NegativeFeedback bool      `json:"negative_feedback"`
}

type DataChunk struct {
	ID            int64     `json:"id"`
	Content       string    `json:"content"`
	Embedding     []float32 `json:"-"` // Don't marshal to JSON response, internal
	EmbeddingJSON string    `json:"-"` // Store as JSON string for DB
}
