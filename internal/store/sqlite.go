package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dataSourceName string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err = store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) initSchema() error {
	schema := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        external_user_id TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS chats (
        id TEXT PRIMARY KEY, -- UUID
        user_id INTEGER NOT NULL,
        title TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id)
    );

    CREATE TABLE IF NOT EXISTS messages (
        id TEXT PRIMARY KEY, -- UUID
        chat_id TEXT NOT NULL,
        sender TEXT NOT NULL CHECK (sender IN ('user', 'model')),
        content TEXT NOT NULL,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
        negative_feedback BOOLEAN DEFAULT FALSE,
        FOREIGN KEY (chat_id) REFERENCES chats (id)
    );

    CREATE TABLE IF NOT EXISTS data_chunks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        content TEXT NOT NULL,
        embedding_json TEXT -- Storing as JSON string of []float32
    );
    `
	_, err := s.db.Exec(schema)
	return err
}

// User methods
func (s *SQLiteStore) GetUserByExternalID(externalUserID string) (*User, error) {
	var user User
	err := s.db.QueryRow("SELECT id, external_user_id, password_hash, created_at FROM users WHERE external_user_id = ?", externalUserID).Scan(&user.ID, &user.ExternalUserID, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	return &user, nil
}

func (s *SQLiteStore) CreateUser(externalUserID, passwordHash string) (*User, error) {
	res, err := s.db.Exec("INSERT INTO users (external_user_id, password_hash) VALUES (?, ?)", externalUserID, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.getUserByID(id)
}

func (s *SQLiteStore) getUserByID(id int64) (*User, error) {
	var user User
	err := s.db.QueryRow("SELECT id, external_user_id, password_hash, created_at FROM users WHERE id = ?", id).Scan(&user.ID, &user.ExternalUserID, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &user, nil
}

// Chat methods
func (s *SQLiteStore) CreateChat(userID int64, title *string) (*Chat, error) {
	chatID := uuid.NewString()
	stmt, err := s.db.Prepare("INSERT INTO chats (id, user_id, title, created_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		return nil, fmt.Errorf("failed to prepare chat insert: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	_, err = stmt.Exec(chatID, userID, title, now)
	if err != nil {
		return nil, fmt.Errorf("failed to execute chat insert: %w", err)
	}
	return &Chat{ID: chatID, UserID: userID, Title: title, CreatedAt: now}, nil
}

func (s *SQLiteStore) GetChatByID(chatID string, userID int64) (*Chat, error) {
	var chat Chat
	var title sql.NullString
	err := s.db.QueryRow("SELECT id, user_id, title, created_at FROM chats WHERE id = ? AND user_id = ?", chatID, userID).Scan(&chat.ID, &chat.UserID, &title, &chat.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}
	if title.Valid {
		chat.Title = &title.String
	}
	return &chat, nil
}

func (s *SQLiteStore) GetChatsByUserID(userID int64) ([]Chat, error) {
	rows, err := s.db.Query("SELECT id, user_id, title, created_at FROM chats WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chats: %w", err)
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var chat Chat
		var title sql.NullString
		if err := rows.Scan(&chat.ID, &chat.UserID, &title, &chat.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chat row: %w", err)
		}
		if title.Valid {
			chat.Title = &title.String
		}
		chats = append(chats, chat)
	}
	return chats, nil
}

func (s *SQLiteStore) UpdateChatTitle(chatID string, userID int64, title string) error {
	stmt, err := s.db.Prepare("UPDATE chats SET title = ? WHERE id = ? AND user_id = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare chat title update: %w", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(title, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to execute chat title update: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("chat not found or not owned by user, title not updated")
	}
	return nil
}

// Message methods
func (s *SQLiteStore) CreateMessage(msg *Message) error {
	msg.ID = uuid.NewString() // Ensure ID is set
	msg.Timestamp = time.Now()

	stmt, err := s.db.Prepare("INSERT INTO messages (id, chat_id, sender, content, timestamp, negative_feedback) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare message insert: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(msg.ID, msg.ChatID, msg.Sender, msg.Content, msg.Timestamp, msg.NegativeFeedback)
	if err != nil {
		return fmt.Errorf("failed to execute message insert: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetMessagesByChatID(chatID string, limit int, offset int) ([]Message, error) {
	query := "SELECT id, chat_id, sender, content, timestamp, negative_feedback FROM messages WHERE chat_id = ? ORDER BY timestamp ASC LIMIT ? OFFSET ?"
	rows, err := s.db.Query(query, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.Sender, &msg.Content, &msg.Timestamp, &msg.NegativeFeedback); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (s *SQLiteStore) GetLastNMessagesByChatID(chatID string, n int) ([]Message, error) {
	query := `
        SELECT id, chat_id, sender, content, timestamp, negative_feedback
        FROM messages
        WHERE chat_id = ?
        ORDER BY timestamp DESC
        LIMIT ?
    `

	rows, err := s.db.Query(query, chatID, n)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.Sender, &msg.Content, &msg.Timestamp, &msg.NegativeFeedback); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *SQLiteStore) UpdateMessageFeedback(messageID string, negativeFeedback bool) error {
	stmt, err := s.db.Prepare("UPDATE messages SET negative_feedback = ? WHERE id = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare feedback update: %w", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(negativeFeedback, messageID)
	if err != nil {
		return fmt.Errorf("failed to execute feedback update: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("message not found, feedback not updated")
	}
	return nil
}

// DataChunk methods (for RAG)
func (s *SQLiteStore) createDataChunk(chunk *DataChunk) error {
	embeddingBytes, err := json.Marshal(chunk.Embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}
	chunk.EmbeddingJSON = string(embeddingBytes)

	stmt, err := s.db.Prepare("INSERT INTO data_chunks (content, embedding_json) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare data_chunk insert: %w", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(chunk.Content, chunk.EmbeddingJSON)
	if err != nil {
		return fmt.Errorf("failed to execute data_chunk insert: %w", err)
	}
	chunk.ID, _ = res.LastInsertId()
	return nil
}

func (s *SQLiteStore) GetAllDataChunks() ([]DataChunk, error) {
	rows, err := s.db.Query("SELECT id, content, embedding_json FROM data_chunks")
	if err != nil {
		return nil, fmt.Errorf("failed to query data_chunks: %w", err)
	}
	defer rows.Close()

	var chunks []DataChunk
	for rows.Next() {
		var chunk DataChunk
		var embeddingJSON string // Read as string from DB
		if err := rows.Scan(&chunk.ID, &chunk.Content, &embeddingJSON); err != nil {
			return nil, fmt.Errorf("failed to scan data_chunk row: %w", err)
		}
		// Ensure embeddingJSON is not empty before trying to unmarshal
		if embeddingJSON != "" {
			if err := json.Unmarshal([]byte(embeddingJSON), &chunk.Embedding); err != nil {
				log.Printf("Warning: failed to unmarshal embedding for chunk %d (content: %.50s...): %v. Embedding will be empty.", chunk.ID, chunk.Content, err)
				// Chunk will have an empty embedding, which might affect similarity search.
				// Consider if this should be a fatal error for the chunk or if it's acceptable.
				chunk.Embedding = nil // Explicitly set to nil if unmarshal fails
			}
		} else {
			log.Printf("Warning: empty embedding_json for chunk ID %d. Embedding will be empty.", chunk.ID)
			chunk.Embedding = nil // Ensure it's nil if the DB field was empty/NULL
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

func (s *SQLiteStore) ClearDataChunks() error {
	_, err := s.db.Exec("DELETE FROM data_chunks")
	if err != nil {
		return fmt.Errorf("failed to delete data_chunks: %w", err)
	}
	_, err = s.db.Exec("DELETE FROM sqlite_sequence WHERE name='data_chunks'")
	if err != nil && !strings.Contains(err.Error(), "no such table") {
		log.Printf("Warning: could not reset sequence for data_chunks: %v", err)
	}
	return nil
}

// IngestDataFromFile reads data.md, extracts text, generates embeddings, and stores them.
func (s *SQLiteStore) IngestDataFromFile(filePath string, embedder func(string) ([]float32, error)) (int, error) {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read data file %s: %w", filePath, err)
	}
	fileContent := string(contentBytes)
	lines := strings.Split(fileContent, "\n")

	var rawChunks []string
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue // Skip empty lines
		}

		// Skip table header and separator
		if i == 0 && strings.Contains(trimmedLine, "|") && (strings.Contains(strings.ToLower(trimmedLine), "text") || strings.Contains(strings.ToLower(trimmedLine), "content")) {
			log.Printf("Skipping table header: %s", trimmedLine)
			continue
		}
		if i == 1 && strings.Contains(trimmedLine, "|") && strings.Contains(trimmedLine, "---") {
			log.Printf("Skipping table separator: %s", trimmedLine)
			continue
		}

		// Basic parsing for a single-column Markdown table row: | some content |
		if strings.HasPrefix(trimmedLine, "|") && strings.HasSuffix(trimmedLine, "|") {
			parts := strings.Split(trimmedLine, "|")
			// Expect 3 parts: "" (before first |), " content ", "" (after last |)
			// Or for | text | header, parts would be ["", " text ", ""]
			if len(parts) >= 3 { // At least | content |
				// The actual content is the second element after splitting by '|', then trim spaces.
				// Example: "| some content |" -> parts are ["", " some content ", ""]
				// Example: "|text|" -> parts are ["", "text", ""]
				// We take parts[1] which is " some content " and trim it.
				cellContent := strings.TrimSpace(parts[1])
				if cellContent != "" {
					rawChunks = append(rawChunks, cellContent)
				} else {
					log.Printf("Skipping row with empty cell content: %s", trimmedLine)
				}
			} else {
				log.Printf("Skipping malformed table row (not enough '|'): %s", trimmedLine)
			}
		} else {
			// If it's not a table row after the header, skip it.
			if i > 1 { // Only log if we're past the typical header/separator lines
				log.Printf("Skipping line not matching table row format: %s", trimmedLine)
			}
		}
	}

	if len(rawChunks) == 0 {
		log.Println("No chunks generated from data file. Ensure it's a Markdown table with a 'text' column and content.")
		return 0, nil // Or an error if this is unexpected
	}

	log.Printf("Generated %d raw chunks from table. Now embedding (this may take a while)...", len(rawChunks))

	if err := s.ClearDataChunks(); err != nil {
		return 0, fmt.Errorf("failed to clear existing data chunks: %w", err)
	}

	count := 0

	ticker := time.NewTicker(40 * time.Millisecond) // delay to not hit rate limit (1500/min)
	defer ticker.Stop()

	for i, rawChunk := range rawChunks {
		<-ticker.C

		embedding, err := embedder(rawChunk)
		if err != nil {
			log.Printf("Failed to generate embedding for chunk %d (\"%.50s...\"): %v. Skipping.", i+1, rawChunk, err)
			continue
		}

		chunk := DataChunk{
			Content:   rawChunk,
			Embedding: embedding,
		}
		if err := s.createDataChunk(&chunk); err != nil {
			log.Printf("Failed to store data chunk %d: %v. Skipping.", i+1, err)
			continue
		}
		count++
		if count%10 == 0 || count == len(rawChunks) {
			log.Printf("Ingested %d/%d chunks...", count, len(rawChunks))
		}
	}
	log.Printf("Successfully ingested %d chunks.", count)
	return count, nil
}
