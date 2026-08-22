package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
)

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
	if version != 14 || dirty {
		t.Fatalf("harap versi 14 dirty=false, dapat %d dirty=%v", version, dirty)
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
