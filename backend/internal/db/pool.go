// Package db holds the database access layer: the connection pool, the
// transaction helper, and the code sqlc generates from db/queries. The
// pool tuning follows research.md R-03 for a 2GB host with Postgres capped at
// 20 connections; 15 for the app leaves 5 for pg_dump, psql, and migrations.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool sizing from research.md R-03. MaxConns 15 stays under the Postgres
// max_connections 20 with headroom for tooling; the rest bound how long idle
// and old connections live so a burst does not pin all 15 forever.
const (
	poolMaxConns          = 15
	poolMinConns          = 2
	poolMaxConnLifetime   = 30 * time.Minute
	poolMaxConnIdleTime   = 5 * time.Minute
	poolHealthCheckPeriod = time.Minute
)

// NewPool opens a tuned pgx pool against databaseURL and verifies it with a
// ping so a bad DSN fails at startup rather than on the first query.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: parse DSN: %w", err)
	}
	cfg.MaxConns = poolMaxConns
	cfg.MinConns = poolMinConns
	cfg.MaxConnLifetime = poolMaxConnLifetime
	cfg.MaxConnIdleTime = poolMaxConnIdleTime
	cfg.HealthCheckPeriod = poolHealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: buka pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}
