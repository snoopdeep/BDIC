// Package db owns the PostgreSQL connection pool and the migration runner.
//
// Migrations are embedded in the binary and applied on start-up inside an
// advisory lock, so two API processes booting at the same moment cannot apply
// the same file twice. There is no separate migration CLI to install.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps the pgx pool so callers depend on this package rather than on pgx
// directly wherever that is practical.
type Pool = pgxpool.Pool

// Open creates the pool and verifies that the database is actually reachable.
// A misconfigured DATABASE_URL should fail here, loudly, at boot — not on the
// first request a parent makes.
func Open(ctx context.Context, databaseURL string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	// Sized for a school, not a social network. A few hundred users, most of
	// them idle most of the day.
	cfg.MaxConns = 12
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 15 * time.Minute
	cfg.HealthCheckPeriod = time.Minute
	cfg.ConnConfig.RuntimeParams["application_name"] = "bdic-api"
	// Every date in this system is an Indian school date. Pinning the session
	// time zone means CURRENT_DATE is today in Chandauli, not today in UTC,
	// which otherwise rolls over at 05:30 local and mis-dates attendance.
	cfg.ConnConfig.RuntimeParams["timezone"] = "Asia/Kolkata"

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres is not responding: %w", err)
	}

	return pool, nil
}
