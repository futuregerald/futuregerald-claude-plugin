package accounts_test

import (
	"context"

	"example.com/user-service/internal/accounts"
)

type fakeStore struct {
	users     map[string]accounts.User
	createErr error // force the failure path
}

func newFakeStore() *fakeStore {
	// The map must be initialised — assigning into a nil map panics.
	return &fakeStore{users: make(map[string]accounts.User)}
}

func (f *fakeStore) ByEmail(_ context.Context, email string) (accounts.User, error) {
	u, ok := f.users[email]
	if !ok {
		return accounts.User{}, accounts.ErrNotFound
	}
	return u, nil
}

func (f *fakeStore) Create(_ context.Context, u accounts.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	// A fake must honour the contract it fakes. The real adapter reports a
	// duplicate email as ErrEmailTaken; a fake that silently overwrote would
	// make the race path untestable and pass tests the real code would fail.
	if _, exists := f.users[u.Email]; exists {
		return accounts.ErrEmailTaken
	}
	f.users[u.Email] = u
	return nil
}
