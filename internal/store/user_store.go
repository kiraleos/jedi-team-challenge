package store

import (
	"database/sql"
	"fmt"
)

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
