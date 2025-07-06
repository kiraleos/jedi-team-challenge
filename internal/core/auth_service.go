package core

import (
	"fmt"
	"gwi.com/jedi-team-challenge/internal/auth"
	"gwi.com/jedi-team-challenge/internal/store"
)

type AuthService struct {
	dbStore store.Storer
}

func NewAuthService(db store.Storer) *AuthService {
	return &AuthService{dbStore: db}
}

type SignupCredentials struct {
	UserID   string
	Password string
}

func (s *AuthService) Signup(creds SignupCredentials) (*store.User, error) {
	if creds.UserID == "" || creds.Password == "" {
		return nil, fmt.Errorf("user ID and password are required")
	}

	hashedPassword, err := auth.HashPassword(creds.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to process password: %w", err)
	}

	user, err := s.dbStore.CreateUser(creds.UserID, hashedPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

type LoginCredentials struct {
	UserID   string
	Password string
}

func (s *AuthService) Login(creds LoginCredentials) (string, error) {
	if creds.UserID == "" || creds.Password == "" {
		return "", fmt.Errorf("user ID and password are required")
	}

	user, err := s.dbStore.GetUserByExternalID(creds.UserID)
	if err != nil {
		// Return a generic error to avoid leaking information about user existence
		return "", fmt.Errorf("invalid credentials")
	}
	if user == nil {
		return "", fmt.Errorf("invalid credentials")
	}

	if !auth.CheckPasswordHash(creds.Password, user.PasswordHash) {
		return "", fmt.Errorf("invalid credentials")
	}

	token, err := auth.GenerateJWT(creds.UserID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}
