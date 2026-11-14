package accounts

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound   = errors.New("accounts: user not found")
	ErrEmailTaken = errors.New("accounts: email already registered")
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// Store is declared here — in the package that calls it, not the one that
// implements it. That is the whole inversion.
type Store interface {
	ByEmail(ctx context.Context, email string) (User, error)
	Create(ctx context.Context, u User) error
}

// Hasher keeps the password-hashing library out of the domain, and keeps the
// algorithm swappable — argon2id today, whatever replaces it later.
type Hasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}
