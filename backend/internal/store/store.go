// Package store holds every SQL query in the application.
//
// Handlers call methods here; they never build SQL themselves. Two reasons:
// a query is easier to review when all of them live together, and the row-level
// scoping that keeps one parent out of another family's records is enforced in
// the query rather than remembered by each handler.
//
// Every parameter is passed as a bind parameter. There is no string
// concatenation of user input into SQL anywhere in this package. Where an
// identifier is a UUID it is cast explicitly with ::uuid so a malformed value
// is rejected by the database rather than silently matching nothing.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the entry point for all database access.
type Store struct {
	pool *pgxpool.Pool
}

// New builds a Store over an existing pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Pool exposes the underlying pool for the few callers that need it, such as
// the audit recorder and the health check.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// ErrNotFound is returned when a query that expected one row found none.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when a write violates a uniqueness rule.
var ErrConflict = errors.New("conflict")

// InTx runs fn inside a transaction, committing on success and rolling back on
// any error or panic.
//
// Used wherever more than one row has to change together: issuing a receipt and
// updating the invoices it settles, converting an application into a student,
// saving a whole class's attendance.
func (s *Store) InTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		// Rollback after a successful commit is a no-op, so this is safe
		// unconditionally and covers the panic path.
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// isUniqueViolation reports whether an error is a Postgres unique-constraint
// violation, and on which constraint. Used to turn a database error into a
// useful message such as "that admission number is already in use".
func isUniqueViolation(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return pgErr.ConstraintName, true
	}
	return "", false
}

// isForeignKeyViolation reports whether an error is a broken reference, which
// usually means an id in the request does not exist.
func isForeignKeyViolation(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return pgErr.ConstraintName, true
	}
	return "", false
}

// isCheckViolation reports whether an error is a CHECK constraint failure,
// which means the database caught something the handler should have.
func isCheckViolation(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23514" {
		return pgErr.ConstraintName, true
	}
	return "", false
}

// noRows maps pgx's sentinel onto this package's.
func noRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
