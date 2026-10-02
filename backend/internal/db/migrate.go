package db

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// advisoryLockKey is an arbitrary but fixed number. Any process running
// migrations takes this lock first, so a second process waits instead of
// applying the same file concurrently.
const advisoryLockKey int64 = 8142026

type migration struct {
	name string
	body string
	hash string
}

// Migrate applies every embedded migration that has not been applied yet, in
// filename order, each one inside its own transaction.
//
// A migration that has already been applied is checked against its recorded
// checksum. If the file has changed since, Migrate refuses to continue: silently
// ignoring an edited migration is how two environments end up with different
// schemas and nobody can tell which is right.
func Migrate(ctx context.Context, pool *Pool, logger *slog.Logger) error {
	pending, err := loadMigrations()
	if err != nil {
		return err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for migrations: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockKey); err != nil {
		return fmt.Errorf("take migration lock: %w", err)
	}
	defer func() {
		// Best effort. If this fails the lock is released when the connection
		// goes back to the pool and is eventually closed.
		_, _ = conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, advisoryLockKey)
	}()

	const createTable = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       text PRIMARY KEY,
			checksum   text        NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`
	if _, err := conn.Exec(ctx, createTable); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := appliedMigrations(ctx, conn.Conn())
	if err != nil {
		return err
	}

	newCount := 0
	for _, item := range pending {
		if existing, ok := applied[item.name]; ok {
			if existing != item.hash {
				return fmt.Errorf(
					"migration %s has changed since it was applied (recorded %s, file %s): "+
						"add a new migration instead of editing an applied one",
					item.name, short(existing), short(item.hash))
			}
			continue
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin %s: %w", item.name, err)
		}
		if _, err := tx.Exec(ctx, item.body); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", item.name, err)
		}
		const record = `INSERT INTO schema_migrations (name, checksum) VALUES ($1, $2)`
		if _, err := tx.Exec(ctx, record, item.name, item.hash); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", item.name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", item.name, err)
		}

		logger.Info("migration applied", "name", item.name)
		newCount++
	}

	if newCount == 0 {
		logger.Info("database schema is up to date", "migrations", len(pending))
	} else {
		logger.Info("database schema updated", "applied", newCount, "total", len(pending))
	}
	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	result := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		body, err := fs.ReadFile(migrationFiles, "migrations/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		sum := sha256.Sum256(body)
		result = append(result, migration{
			name: entry.Name(),
			body: string(body),
			hash: hex.EncodeToString(sum[:]),
		})
	}

	if len(result) == 0 {
		return nil, errors.New("no migrations found: the binary was built without migrations/*.sql")
	}

	sort.Slice(result, func(i, j int) bool { return result[i].name < result[j].name })
	return result, nil
}

func appliedMigrations(ctx context.Context, conn *pgx.Conn) (map[string]string, error) {
	resultRows, err := conn.Query(ctx, `SELECT name, checksum FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer resultRows.Close()

	applied := make(map[string]string)
	for resultRows.Next() {
		var name, checksum string
		if err := resultRows.Scan(&name, &checksum); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[name] = checksum
	}
	if err := resultRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema_migrations: %w", err)
	}
	return applied, nil
}

func short(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}
