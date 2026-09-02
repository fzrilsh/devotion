package masterdata

import (
	"context"
	"fmt"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
)

// baselineProducts is the starter list of garment product types a subcontractor
// picks from when creating a listing (FR-058). No source document pins the
// exact set, so this is a sensible baseline for Indonesian konveksi work; admins
// extend it later through the proposal flow. Names are user-facing, so they are
// Indonesian. Order here is the display sort_order.
var baselineProducts = []string{
	"Kaos",
	"Kemeja",
	"Kaos Polo",
	"Jaket",
	"Hoodie",
	"Celana",
	"Seragam",
	"Jersey",
	"Gamis",
	"Kerudung",
	"Tas Kain",
	"Topi",
}

// baselineMachines is the starter list of machine types. The set mirrors the
// stations a small konveksi runs. Same rules as baselineProducts: Indonesian
// display names, slice order is the display order.
var baselineMachines = []string{
	"Mesin Jahit Jarum 1",
	"Mesin Jahit Jarum 2",
	"Mesin Obras",
	"Mesin Overdeck",
	"Mesin Rantai",
	"Mesin Bordir",
	"Mesin Potong Kain",
	"Mesin Kancing",
	"Mesin Lubang Kancing",
	"Mesin Press",
}

// SeedMasterData upserts the baseline product and machine catalog. It is
// idempotent on (type, name): a second run reactivates and reorders existing
// rows without duplicating, and never deletes, so an admin's later additions
// survive a reseed. created_at comes from the Clock, since the column has no DB
// default (Rule 5, no time.Now in business logic).
func (s *Service) SeedMasterData(ctx context.Context) error {
	q := s.queries()
	now := tstz(s.clock.Now())

	for i, name := range baselineProducts {
		if err := q.UpsertCatalogItem(ctx, sqlcgen.UpsertCatalogItemParams{
			Type:      sqlcgen.ItemTypeProduct,
			Name:      name,
			SortOrder: int32(i),
			CreatedAt: now,
		}); err != nil {
			return fmt.Errorf("upsert produk %q: %w", name, err)
		}
	}
	for i, name := range baselineMachines {
		if err := q.UpsertCatalogItem(ctx, sqlcgen.UpsertCatalogItemParams{
			Type:      sqlcgen.ItemTypeMachine,
			Name:      name,
			SortOrder: int32(i),
			CreatedAt: now,
		}); err != nil {
			return fmt.Errorf("upsert mesin %q: %w", name, err)
		}
	}
	return nil
}
