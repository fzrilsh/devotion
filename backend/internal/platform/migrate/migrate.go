// Package migrate applies the embedded SQL migrations at startup. It guards the
// run with pg_try_advisory_lock so that when two containers overlap during a
// deploy rollover the second one skips rather than blocking: blocking would
// hold up the rollover, and the migrations are identical either way.
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"

	devotiondb "github.com/fzrilsh/devotion/backend/db"
)

// advisoryLockKey is a fixed application-chosen key. Every process uses the
// same literal so they contend on one lock; the value itself is arbitrary.
const advisoryLockKey int64 = 5470130124100001

// migrationsSubdir is the directory inside the embedded FS holding the files.
const migrationsSubdir = "migrations"

// Run applies all pending migrations against databaseURL. It returns nil
// without error when another process already holds the advisory lock, since
// that process is applying the same set.
func Run(ctx context.Context, databaseURL string, log *slog.Logger) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("migrate: buka koneksi: %w", err)
	}
	defer db.Close()

	// Pin a single connection for the advisory lock: a session-scoped lock
	// released on a different connection is never released.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migrate: ambil koneksi: %w", err)
	}
	defer conn.Close()

	var acquired bool
	if err := conn.QueryRowContext(ctx,
		"SELECT pg_try_advisory_lock($1)", advisoryLockKey).Scan(&acquired); err != nil {
		return fmt.Errorf("migrate: pg_try_advisory_lock: %w", err)
	}
	if !acquired {
		log.Info("migrasi dilewati: proses lain sedang memigrasi")
		return nil
	}
	defer func() {
		if _, err := conn.ExecContext(context.Background(),
			"SELECT pg_advisory_unlock($1)", advisoryLockKey); err != nil {
			log.Error("melepas advisory lock migrasi gagal", "error", err)
		}
	}()

	src, err := iofs.New(devotiondb.MigrationsFS, migrationsSubdir)
	if err != nil {
		return fmt.Errorf("migrate: baca berkas migrasi: %w", err)
	}
	defer src.Close()

	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("migrate: driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("migrate: inisialisasi: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: jalankan: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("migrate: versi: %w", err)
	}
	log.Info("migrasi selesai", "version", version, "dirty", dirty)
	return nil
}

// newMigrator builds a *migrate.Migrate over the embedded files against dsn.
// It opens its own connection; the caller must Close it. The pgx/v5 database
// driver registers under the "pgx5" scheme, so the DSN scheme is rewritten.
func newMigrator(dsn string) (*migrate.Migrate, error) {
	src, err := iofs.New(devotiondb.MigrationsFS, migrationsSubdir)
	if err != nil {
		return nil, fmt.Errorf("migrate: baca berkas migrasi: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, toPgx5Scheme(dsn))
	if err != nil {
		return nil, fmt.Errorf("migrate: inisialisasi: %w", err)
	}
	return m, nil
}

// toPgx5Scheme swaps a postgres:// or postgresql:// scheme for pgx5:// so the
// golang-migrate driver registry resolves the pgx/v5 driver.
func toPgx5Scheme(dsn string) string {
	for _, p := range []string{"postgresql://", "postgres://"} {
		if len(dsn) >= len(p) && dsn[:len(p)] == p {
			return "pgx5://" + dsn[len(p):]
		}
	}
	return dsn
}

// ensure fs.FS is referenced so the import stays even if the embed API shifts.
var _ fs.FS = devotiondb.MigrationsFS
