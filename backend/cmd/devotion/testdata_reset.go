package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// resetFixtureTables lists every table the seeder writes, in an order that does
// not matter because the TRUNCATE below cascades. reset:test-data empties them
// so seed:test-data can run again from a clean slate. Region and catalog rows
// (province, city, catalog_item) are left in place: ensurePrerequisites upserts
// them idempotently and seed:regions or seed:master-data may own them, so reset
// does not claim them.
var resetFixtureTables = []string{
	"capacity_allocation",
	"payment_record",
	"review",
	"dispute",
	"work_order_status_history",
	"work_order",
	"offer",
	"request_candidate",
	"quota_request",
	"verification_request",
	"uploaded_file",
	"availability_period",
	"listing_product",
	"listing_machine",
	"capacity_listing",
	"notification_channel",
	"notification",
	"item_proposal",
	"business_profile",
	"session",
	"user_account",
}

// resetTestData empties every table the seeder writes so seed:test-data starts
// from a clean slate. It runs one TRUNCATE ... CASCADE covering all fixture
// tables, so the child rows drop with their parents regardless of the RESTRICT
// foreign keys that would block a piecewise delete. The production guard in
// openTestDataPool has already run, so this never reaches a real database
// (T075). Region and catalog rows are deliberately excluded, see
// resetFixtureTables.
func resetTestData(ctx context.Context, pool *pgxpool.Pool) error {
	stmt := "TRUNCATE TABLE " + strings.Join(resetFixtureTables, ", ") + " RESTART IDENTITY CASCADE"
	if _, err := pool.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("reset test data: %w", err)
	}
	return nil
}
