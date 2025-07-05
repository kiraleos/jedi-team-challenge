package store

// Storer defines the interface for database operations required by services.
// This allows for mocking the store in tests.
type Storer interface {
	GetUserByExternalID(externalUserID string) (*User, error)
	CreateUser(externalUserID, passwordHash string) (*User, error)
}
