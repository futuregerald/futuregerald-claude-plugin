package accounts

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Service struct {
	store  Store
	hasher Hasher
	now    func() time.Time
	newID  func() string
}

func NewService(store Store, hasher Hasher, now func() time.Time, newID func() string) *Service {
	return &Service{store: store, hasher: hasher, now: now, newID: newID}
}

func (s *Service) Register(ctx context.Context, email, password string) (User, error) {
	// This pre-check exists to return a clean error in the common case. It is
	// NOT the uniqueness guarantee — two concurrent calls can both pass it.
	// The unique index on users.email is the guarantee, and Store.Create
	// reports a violation as ErrEmailTaken. Both paths are handled.
	if _, err := s.store.ByEmail(ctx, email); err == nil {
		return User{}, ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return User{}, fmt.Errorf("lookup existing: %w", err)
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	u := User{
		ID:           s.newID(),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    s.now(),
	}
	if err := s.store.Create(ctx, u); err != nil {
		// Already wrapped as ErrEmailTaken by the adapter when the unique
		// index fires; errors.Is at the transport edge still matches.
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}
