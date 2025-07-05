package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gwi.com/jedi-team-challenge/internal/auth"
	"gwi.com/jedi-team-challenge/internal/store"
)

// MockStorer is a mock implementation of the Storer interface.
type MockStorer struct {
	mock.Mock
}

// GetUserByExternalID is a mock method.
func (m *MockStorer) GetUserByExternalID(externalUserID string) (*store.User, error) {
	args := m.Called(externalUserID)
	// Handle the case where the first return value is nil.
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.User), args.Error(1)
}

// CreateUser is a mock method.
func (m *MockStorer) CreateUser(externalUserID, passwordHash string) (*store.User, error) {
	args := m.Called(externalUserID, passwordHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.User), args.Error(1)
}

// --- AuthService Tests ---

func TestAuthService_Signup(t *testing.T) {
	mockStore := new(MockStorer)
	authService := NewAuthService(mockStore)

	t.Run("Successful Signup", func(t *testing.T) {
		creds := SignupCredentials{
			UserID:   "testuser",
			Password: "password123",
		}

		// We need to assert that the password hash will be checked with the correct password
		hashedPassword, _ := auth.HashPassword(creds.Password)

		mockStore.On("CreateUser", creds.UserID, mock.AnythingOfType("string")).Return(&store.User{
			ID:             1,
			ExternalUserID: creds.UserID,
			PasswordHash:   hashedPassword,
			CreatedAt:      time.Now(),
		}, nil).Once()

		user, err := authService.Signup(creds)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "testuser", user.ExternalUserID)
		mockStore.AssertExpectations(t)
	})

	t.Run("Signup with empty credentials", func(t *testing.T) {
		creds := SignupCredentials{UserID: "", Password: ""}
		_, err := authService.Signup(creds)

		assert.Error(t, err)
		assert.Equal(t, "user ID and password are required", err.Error())
	})

	t.Run("Signup with database error", func(t *testing.T) {
		creds := SignupCredentials{
			UserID:   "testuser",
			Password: "password123",
		}

		mockStore.On("CreateUser", creds.UserID, mock.AnythingOfType("string")).Return(nil, fmt.Errorf("database error")).Once()

		_, err := authService.Signup(creds)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create user")
		mockStore.AssertExpectations(t)
	})
}

func TestAuthService_Login(t *testing.T) {
	mockStore := new(MockStorer)
	authService := NewAuthService(mockStore)

	hashedPassword, _ := auth.HashPassword("password123")
	existingUser := &store.User{
		ID:             1,
		ExternalUserID: "testuser",
		PasswordHash:   hashedPassword,
	}

	t.Run("Successful Login", func(t *testing.T) {
		loginCreds := LoginCredentials{
			UserID:   "testuser",
			Password: "password123",
		}

		mockStore.On("GetUserByExternalID", "testuser").Return(existingUser, nil).Once()

		token, err := authService.Login(loginCreds)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		mockStore.AssertExpectations(t)
	})

	t.Run("Login with invalid password", func(t *testing.T) {
		loginCreds := LoginCredentials{
			UserID:   "testuser",
			Password: "wrongpassword",
		}

		mockStore.On("GetUserByExternalID", "testuser").Return(existingUser, nil).Once()

		_, err := authService.Login(loginCreds)

		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
		mockStore.AssertExpectations(t)
	})

	t.Run("Login with non-existent user", func(t *testing.T) {
		loginCreds := LoginCredentials{
			UserID:   "nonexistent",
			Password: "password123",
		}

		mockStore.On("GetUserByExternalID", "nonexistent").Return(nil, nil).Once()

		_, err := authService.Login(loginCreds)

		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
		mockStore.AssertExpectations(t)
	})

	t.Run("Login with database error", func(t *testing.T) {
		loginCreds := LoginCredentials{
			UserID:   "testuser",
			Password: "password123",
		}

		mockStore.On("GetUserByExternalID", "testuser").Return(nil, fmt.Errorf("database error")).Once()

		_, err := authService.Login(loginCreds)

		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
		mockStore.AssertExpectations(t)
	})
}
