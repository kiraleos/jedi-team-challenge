package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

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
