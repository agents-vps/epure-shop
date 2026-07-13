package sqlite

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/agents-vps/epure-shop/internal/core/ports"
)

// BcryptHasher implements ports.PasswordHasher using bcrypt.
type BcryptHasher struct{ cost int }

// NewBcryptHasher returns a bcrypt hasher with the recommended cost (12).
func NewBcryptHasher() ports.PasswordHasher {
	return &BcryptHasher{cost: 12}
}

// Hash produces a bcrypt hash from a plaintext password.
func (h *BcryptHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Verify compares a plaintext password against a bcrypt hash.
func (h *BcryptHasher) Verify(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
