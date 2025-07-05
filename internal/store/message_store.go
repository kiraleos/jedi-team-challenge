package store

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

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
