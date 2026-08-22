package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTx runs fn inside a transaction. It rolls back on error and on panic,
// then re-panics: a panic in the middle of a capacity allocation must not
// leave a transaction open holding row locks. On success it commits.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db: mulai transaksi: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			// Roll back before re-panicking so the connection is not returned
			// to the pool with an open transaction.
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr != pgx.ErrTxClosed {
				err = fmt.Errorf("%w; rollback: %v", err, rbErr)
			}
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("db: commit: %w", err)
	}
	return nil
}
