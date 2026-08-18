package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
)

type txKey struct{}

// querier is satisfied by both *sql.DB and *sql.Tx, so store methods do not
// care whether they are inside a transaction.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type TxManager struct{ db *sql.DB }

func NewTxManager(db *sql.DB) *TxManager { return &TxManager{db: db} }

func (m *TxManager) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	// Rollback after a successful Commit is a no-op, so this is safe
	// unconditionally and covers panics and early returns.
	defer func() { _ = tx.Rollback() }()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// q returns the transaction bound to ctx if there is one, else the pool.
// Every store method uses this instead of touching s.db directly.
func (s *AccountStore) q(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return s.db
}
