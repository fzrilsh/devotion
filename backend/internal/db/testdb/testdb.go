// Package testdb gives tests a migrated, isolated Postgres schema on the shared
// database, honoring the no-extra-service rule: each test gets its own
// test_<name> schema with search_path set, migrated once, and truncated on
// cleanup so a rerun skips re-migrating. When DATABASE_URL_TEST is unreachable
// the helper skips while naming the variable, so a missing database never looks
// like a passing suite.
package testdb

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/platform/migrate"
)

const defaultDSN = "postgres://devotion:devotion@127.0.0.1:5432/devotion?sslmode=disable"

// quietLogger discards the migrate runner's output during tests.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// New creates (or resets) schema test_<name>, migrates it, and returns a pool
// pinned to search_path=test_<name>,public. public must stay on the path
// because citext and pgcrypto are database-scoped, not schema-scoped. The pool
// is truncated and closed on t.Cleanup.
func New(t *testing.T, name string) *pgxpool.Pool {
	t.Helper()
	base := baseDSN(t)
	schema := "test_" + name
	ctx := context.Background()

	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("testdb: connect: %v", err)
	}
	defer admin.Close(ctx)

	if _, err := admin.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema)); err != nil {
		t.Fatalf("testdb: drop schema: %v", err)
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema)); err != nil {
		t.Fatalf("testdb: create schema: %v", err)
	}

	schemaURL := withSearchPath(t, base, schema)
	if err := migrate.Run(ctx, schemaURL, quietLogger()); err != nil {
		t.Fatalf("testdb: migrasi: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(schemaURL)
	if err != nil {
		t.Fatalf("testdb: parse pool DSN: %v", err)
	}
	cfg.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("testdb: pool: %v", err)
	}

	t.Cleanup(func() {
		truncateAll(context.Background(), pool, schema)
		pool.Close()
	})
	return pool
}

// baseDSN returns DATABASE_URL_TEST or the compose default, skipping the test
// (naming the variable) when the database cannot be reached.
func baseDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		dsn = defaultDSN
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("DATABASE_URL_TEST tidak terjangkau (%v); lewati uji yang butuh database", err)
	}
	conn.Close(ctx)
	return dsn
}

// withSearchPath appends search_path=<schema>,public as a query parameter so
// pgx forwards it as a connection option for every connection in the pool.
func withSearchPath(t *testing.T, base, schema string) string {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("testdb: parse DSN: %v", err)
	}
	q := u.Query()
	q.Set("search_path", schema+",public")
	u.RawQuery = q.Encode()
	return u.String()
}

// truncateAll empties every base table in the schema so a rerun reuses the
// already-migrated schema instead of dropping and re-migrating it.
func truncateAll(ctx context.Context, pool *pgxpool.Pool, schema string) {
	rows, err := pool.Query(ctx, `
		SELECT tablename FROM pg_tables
		WHERE schemaname = $1 AND tablename <> 'schema_migrations'`, schema)
	if err != nil {
		return
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return
		}
		tables = append(tables, fmt.Sprintf("%s.%s", schema, name))
	}
	rows.Close()
	if len(tables) == 0 {
		return
	}
	_, _ = pool.Exec(ctx, "TRUNCATE "+strings.Join(tables, ", ")+" RESTART IDENTITY CASCADE")
}
