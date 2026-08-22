package migrate

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

// testDSN returns the test database URL or skips. DATABASE_URL_TEST defaults to
// the compose Postgres; when it cannot be reached the test skips while naming
// the variable, so a missing database never looks like a passing suite.
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		dsn = "postgres://devotion:devotion@localhost:5432/devotion?sslmode=disable"
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("DATABASE_URL_TEST tidak terjangkau (%v); lewati uji integrasi migrasi", err)
	}
	conn.Close(ctx)
	return dsn
}

// schemaDSN adds a dedicated search_path so each test isolates its migration
// state in its own schema on the shared Postgres, per the no-extra-service rule.
func schemaDSN(t *testing.T, base, schema string) string {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	q := u.Query()
	q.Set("search_path", schema+",public")
	u.RawQuery = q.Encode()
	return u.String()
}

func setupSchema(t *testing.T, dsn, schema string) *pgx.Conn {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema)); err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schema)); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	t.Cleanup(func() {
		c, err := pgx.Connect(context.Background(), dsn)
		if err == nil {
			c.Exec(context.Background(), fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
			c.Close(context.Background())
		}
		conn.Close(context.Background())
	})
	return conn
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestRun_ReachesVersion14Clean verifies the full stack applies and lands at
// version 15 with dirty=false, and that a second run is a no-op.
func TestRun_ReachesVersion14Clean(t *testing.T) {
	base := testDSN(t)
	const schema = "test_migrate_v14"
	conn := setupSchema(t, base, schema)
	dsn := schemaDSN(t, base, schema)
	ctx := context.Background()

	if err := Run(ctx, dsn, quietLogger()); err != nil {
		t.Fatalf("run pertama: %v", err)
	}

	var version int
	var dirty bool
	if err := conn.QueryRow(ctx,
		fmt.Sprintf("SELECT version, dirty FROM %s.schema_migrations", schema)).
		Scan(&version, &dirty); err != nil {
		t.Fatalf("baca schema_migrations: %v", err)
	}
	if version != 15 || dirty {
		t.Fatalf("harap versi 15 dirty=false, dapat versi %d dirty=%v", version, dirty)
	}

	// Idempotent: a second run changes nothing and does not error.
	if err := Run(ctx, dsn, quietLogger()); err != nil {
		t.Fatalf("run kedua (idempoten): %v", err)
	}
}

// TestRun_DownUpReturnsSameVersion verifies migrating fully down then back up
// lands on version 15 again, exercising the exact-reverse down migrations.
func TestRun_DownUpReturnsSameVersion(t *testing.T) {
	base := testDSN(t)
	const schema = "test_migrate_downup"
	conn := setupSchema(t, base, schema)
	dsn := schemaDSN(t, base, schema)
	ctx := context.Background()

	if err := Run(ctx, dsn, quietLogger()); err != nil {
		t.Fatalf("run awal: %v", err)
	}
	m, err := newMigrator(dsn)
	if err != nil {
		t.Fatalf("migrator: %v", err)
	}
	defer m.Close()
	if err := m.Down(); err != nil {
		t.Fatalf("down: %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("up: %v", err)
	}
	var version int
	var dirty bool
	if err := conn.QueryRow(ctx,
		fmt.Sprintf("SELECT version, dirty FROM %s.schema_migrations", schema)).
		Scan(&version, &dirty); err != nil {
		t.Fatalf("baca schema_migrations: %v", err)
	}
	if version != 15 || dirty {
		t.Fatalf("setelah down-up harap versi 15 dirty=false, dapat %d dirty=%v", version, dirty)
	}
}

// TestMigrations_TriggersInstalled checks the three trigger functions produce
// their triggers on the expected tables.
func TestMigrations_TriggersInstalled(t *testing.T) {
	base := testDSN(t)
	const schema = "test_migrate_triggers"
	conn := setupSchema(t, base, schema)
	dsn := schemaDSN(t, base, schema)
	ctx := context.Background()
	if err := Run(ctx, dsn, quietLogger()); err != nil {
		t.Fatalf("run: %v", err)
	}

	wantTriggers := []string{
		"trg_reject_wrong_product_item",
		"trg_reject_wrong_machine_item",
		"trg_reject_self_request",
		"trg_reject_allocation_before_readiness",
	}
	for _, name := range wantTriggers {
		var count int
		if err := conn.QueryRow(ctx, `
			SELECT count(*) FROM pg_trigger tg
			JOIN pg_class c ON c.oid = tg.tgrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = $1 AND tg.tgname = $2 AND NOT tg.tgisinternal`,
			schema, name).Scan(&count); err != nil {
			t.Fatalf("query pg_trigger %s: %v", name, err)
		}
		if count != 1 {
			t.Errorf("trigger %s: harap 1 terpasang, dapat %d", name, count)
		}
	}
}

// TestMigrations_KeyConstraints checks the four constraints the plan singles out
// exist via pg_constraint.
func TestMigrations_KeyConstraints(t *testing.T) {
	base := testDSN(t)
	const schema = "test_migrate_constraints"
	conn := setupSchema(t, base, schema)
	dsn := schemaDSN(t, base, schema)
	ctx := context.Background()
	if err := Run(ctx, dsn, quietLogger()); err != nil {
		t.Fatalf("run: %v", err)
	}

	wantConstraints := []string{
		"used_capacity_within_total",
		"week_start_is_monday",
		"readiness_not_past_deadline",
		"city_belongs_to_province",
	}
	for _, name := range wantConstraints {
		var count int
		if err := conn.QueryRow(ctx, `
			SELECT count(*) FROM pg_constraint con
			JOIN pg_namespace n ON n.oid = con.connamespace
			WHERE n.nspname = $1 AND con.conname = $2`,
			schema, name).Scan(&count); err != nil {
			t.Fatalf("query pg_constraint %s: %v", name, err)
		}
		if count < 1 {
			t.Errorf("constraint %s tidak ditemukan", name)
		}
	}
}
