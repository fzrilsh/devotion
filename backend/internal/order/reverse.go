package order

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
)

// ReverseAllocation undoes the whole capacity allocation of one work order in a
// single transaction (FR-020). It returns every period's used_capacity to the
// value it held before the order formed and marks each allocation row
// reversed_at without deleting it, so the audit trail survives. The locking
// mirrors formation (R-04): lock the work order, then its active allocation
// periods ascending by week_start, the deadlock preventer. Already reversed
// rows are skipped, so a repeat call is a no-op rather than a double refund.
//
// This is the reversal capability US5's cancellation (T054) builds on. It does
// not itself change the work order status or write history; the cancellation
// path owns those. For T048 the reversal opens its own transaction and stays
// self-contained.
func (s *Service) ReverseAllocation(ctx context.Context, workOrderID pgtype.UUID) error {
	now := s.clock.Now()
	return db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		// Lock the work order first, in the same spirit formation locks the listing,
		// so a concurrent reversal or status change on the same order serializes.
		if _, err := q.LockWorkOrderForReversal(ctx, workOrderID); err != nil {
			return err
		}

		// Lock the active allocation rows and their periods ascending by week_start.
		// The order matches LockPeriodsInRange in formation: two transactions on the
		// same periods always take them in the same order (R-04 deadlock preventer).
		allocs, err := q.ListActiveAllocationsForReversal(ctx, workOrderID)
		if err != nil {
			return err
		}

		for _, a := range allocs {
			// Return the period to its pre-order used_capacity. The amount is the
			// allocation's own quantity, so the subtraction cannot drop below zero.
			if _, err := q.LowerUsedCapacity(ctx, sqlcgen.LowerUsedCapacityParams{
				ID:           a.PeriodID,
				UsedCapacity: a.Quantity,
				UpdatedAt:    tstz(now),
			}); err != nil {
				return err
			}
			// Mark the row reversed rather than deleting it (FR-020).
			if err := q.MarkAllocationReversed(ctx, sqlcgen.MarkAllocationReversedParams{
				ID:         a.AllocationID,
				ReversedAt: tstz(now),
			}); err != nil {
				return err
			}
		}
		return nil
	})
}
