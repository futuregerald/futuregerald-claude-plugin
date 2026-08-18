package accounts_test

// Exercises the doc's accounts.Service with the doc's own fakeStore.
// No database. Verifies the finding-1 fix (ID is actually populated) and
// that injected now/newID make the result deterministic.

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/user-service/internal/accounts"
)

type stubHasher struct{ err error }

func (s stubHasher) Hash(plain string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return "hashed:" + plain, nil
}

func (s stubHasher) Compare(hash, plain string) error { return nil }

var fixedTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func newSvc(store accounts.Store) *accounts.Service {
	return accounts.NewService(
		store,
		stubHasher{},
		func() time.Time { return fixedTime },
		func() string { return "user-1" },
	)
}

func TestRegisterPopulatesID(t *testing.T) {
	u, err := newSvc(newFakeStore()).Register(context.Background(), "a@example.com", "pw")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	// This is the regression the reviewer found: ID used to come back "".
	if u.ID != "user-1" {
		t.Errorf("ID = %q, want %q", u.ID, "user-1")
	}
	if u.Email != "a@example.com" {
		t.Errorf("Email = %q", u.Email)
	}
	if u.PasswordHash != "hashed:pw" {
		t.Errorf("PasswordHash = %q", u.PasswordHash)
	}
	if !u.CreatedAt.Equal(fixedTime) {
		t.Errorf("CreatedAt = %v, want %v", u.CreatedAt, fixedTime)
	}
}

func TestRegisterDuplicateViaPreCheck(t *testing.T) {
	store := newFakeStore()
	svc := newSvc(store)

	if _, err := svc.Register(context.Background(), "dup@example.com", "pw"); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	_, err := svc.Register(context.Background(), "dup@example.com", "pw")
	if !errors.Is(err, accounts.ErrEmailTaken) {
		t.Errorf("second Register err = %v, want ErrEmailTaken", err)
	}
}

func TestRegisterPropagatesStoreErrorWrapped(t *testing.T) {
	boom := errors.New("disk on fire")
	store := newFakeStore()
	store.createErr = boom

	_, err := newSvc(store).Register(context.Background(), "x@example.com", "pw")
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want it to wrap %v", err, boom)
	}
}

// The adapter reports a unique-index violation as ErrEmailTaken. The service
// wraps it with %w, so errors.Is must still match at the transport edge --
// that is what makes the 409 mapping work under a race.
func TestRegisterSurfacesAdapterEmailTakenThroughWrap(t *testing.T) {
	store := newFakeStore()
	store.createErr = accounts.ErrEmailTaken

	_, err := newSvc(store).Register(context.Background(), "race@example.com", "pw")
	if !errors.Is(err, accounts.ErrEmailTaken) {
		t.Errorf("err = %v, want errors.Is(..., ErrEmailTaken) to hold through the wrap", err)
	}
}
