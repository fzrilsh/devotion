package db

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
)

// highestMigrationVersion reads backend/db/migrations and returns the largest
// numeric prefix present, so this test asserts the schema lands on whatever is
// on disk instead of a hardcoded number. Adding a migration does not force an
// edit here; the test still fails if the applied version does not match disk.
func highestMigrationVersion(t *testing.T) int {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			dir = filepath.Join(dir, "db", "migrations")
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod tidak ditemukan di atas direktori kerja")
		}
		dir = parent
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("baca direktori migrasi: %v", err)
	}
	num := regexp.MustCompile(`^(\d{6})_.*\.up\.sql$`)
	highest := 0
	for _, e := range entries {
		m := num.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("nomor migrasi tidak sah pada %s: %v", e.Name(), err)
		}
		if n > highest {
			highest = n
		}
	}
	if highest == 0 {
		t.Fatal("tidak ada berkas migrasi up ditemukan")
	}
	return highest
}

// TestPool_MigratedSchemaAndPing verifies the testdb harness migrates a fresh
// schema and the generated Ping query runs through the pool.
func TestPool_MigratedSchemaAndPing(t *testing.T) {
	pool := testdb.New(t, "pool_ping")
	ctx := context.Background()

	q := sqlcgen.New(pool)
	got, err := q.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if got != 1 {
		t.Fatalf("Ping = %d, mau 1", got)
	}

	var version int
	var dirty bool
	if err := pool.QueryRow(ctx,
		"SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil {
		t.Fatalf("baca schema_migrations: %v", err)
	}
	want := highestMigrationVersion(t)
	if version != want || dirty {
		t.Fatalf("harap versi %d dirty=false, dapat %d dirty=%v", want, version, dirty)
	}
}

// TestWithTx_RollsBackOnError checks a returned error rolls the transaction
// back so no partial write survives.
func TestWithTx_RollsBackOnError(t *testing.T) {
	pool := testdb.New(t, "withtx_rollback")
	ctx := context.Background()

	wantErr := errors.New("gagal sengaja")
	err := WithTx(ctx, pool, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx,
			`INSERT INTO province (code, name) VALUES ('99', 'Uji')`); e != nil {
			return e
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("WithTx error = %v, mau %v", err, wantErr)
	}

	var count int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM province WHERE code = '99'").Scan(&count); err != nil {
		t.Fatalf("hitung province: %v", err)
	}
	if count != 0 {
		t.Fatalf("baris seharusnya di-rollback, dapat %d baris", count)
	}
}

// TestWithTx_CommitsOnSuccess checks a nil return commits the write.
func TestWithTx_CommitsOnSuccess(t *testing.T) {
	pool := testdb.New(t, "withtx_commit")
	ctx := context.Background()

	if err := WithTx(ctx, pool, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx,
			`INSERT INTO province (code, name) VALUES ('98', 'Uji Commit')`)
		return e
	}); err != nil {
		t.Fatalf("WithTx: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM province WHERE code = '98'").Scan(&count); err != nil {
		t.Fatalf("hitung province: %v", err)
	}
	if count != 1 {
		t.Fatalf("baris seharusnya ter-commit, dapat %d baris", count)
	}
}
