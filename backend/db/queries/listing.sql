-- Capacity listing and its availability calendar. One listing per profile
-- (one_listing_per_profile); the weekly capacity is the central number
-- (FR-012/FR-014) propagated to future periods on edit (FR-089). Periods are
-- stored as the Monday date of each week (week_start_is_monday); the horizon is
-- the furthest Monday that already has a period.

-- CreateListing inserts the single listing a profile may own. horizon_until is
-- seeded to the current week's Monday (it is NOT NULL and must be a Monday);
-- EnsureHorizon raises it and generates the calendar in the same transaction.
-- calendar_updated_at starts at creation time; only an owner edit to the
-- calendar advances it afterward (FR-021).
-- name: CreateListing :one
INSERT INTO capacity_listing (
    id, profile_id, weekly_capacity, readiness_lead_days,
    published, calendar_updated_at, horizon_until, created_at, updated_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, true, $4, $5, $4, $4
)
RETURNING *;

-- GetListingByProfile loads a profile's listing, backing GET /listing/me. A
-- missing row is a genuine 404 (the profile has not created a listing yet), not
-- an invariant violation.
-- name: GetListingByProfile :one
SELECT * FROM capacity_listing WHERE profile_id = $1;

-- GetListingByID loads any listing by its own id, for the public profile join.
-- name: GetListingByID :one
SELECT * FROM capacity_listing WHERE id = $1;

-- LockListingByProfile takes a row lock on the profile's listing so a capacity
-- edit and its propagation to future periods run as one serialized unit. The
-- listing row is always locked before any availability_period row, and that
-- order is fixed across the codebase to prevent deadlocks.
-- name: LockListingByProfile :one
SELECT * FROM capacity_listing WHERE profile_id = $1 FOR UPDATE;

-- UpdateListing writes the owner-editable listing fields. Publication state is
-- changed through SetListingPublished, not here, so a capacity edit never flips
-- visibility by accident.
-- name: UpdateListing :one
UPDATE capacity_listing
SET weekly_capacity     = $2,
    readiness_lead_days = $3,
    updated_at          = $4
WHERE id = $1
RETURNING *;

-- SetListingPublished toggles visibility for PUT /listing/me/visibility. A
-- hidden listing keeps its calendar and allocations; it simply drops out of
-- search until re-enabled.
-- name: SetListingPublished :one
UPDATE capacity_listing
SET published = $2, updated_at = $3
WHERE id = $1
RETURNING *;

-- TouchCalendarUpdatedAt stamps the calendar as freshly edited. This is the
-- only path that advances calendar_updated_at, so a listing whose owner never
-- edits the calendar reads as stale and FR-021 keeps working.
-- name: TouchCalendarUpdatedAt :exec
UPDATE capacity_listing
SET calendar_updated_at = $2, updated_at = $2
WHERE id = $1;

-- RaiseHorizonUntil moves the horizon forward, never back: GREATEST keeps the
-- result independent of commit order when two requests extend it at once.
-- name: RaiseHorizonUntil :exec
UPDATE capacity_listing
SET horizon_until = GREATEST(horizon_until, $2), updated_at = $3
WHERE id = $1;

-- InsertListingProduct links a product item to the listing. The
-- trg_reject_wrong_product_item trigger rejects a machine id here, but the
-- service checks the item type first so a mismatch is a 422, not a 500.
-- name: InsertListingProduct :exec
INSERT INTO listing_product (listing_id, item_id) VALUES ($1, $2);

-- DeleteListingProducts clears a listing's product links before rewriting them
-- on an edit.
-- name: DeleteListingProducts :exec
DELETE FROM listing_product WHERE listing_id = $1;

-- InsertListingMachine links a machine item and its unit count to the listing.
-- name: InsertListingMachine :exec
INSERT INTO listing_machine (listing_id, item_id, machine_count) VALUES ($1, $2, $3);

-- DeleteListingMachines clears a listing's machine links before rewriting them.
-- name: DeleteListingMachines :exec
DELETE FROM listing_machine WHERE listing_id = $1;

-- ListListingProducts returns the product items of a listing joined to their
-- catalog rows, so the response carries names, not bare ids.
-- name: ListListingProducts :many
SELECT i.id, i.type, i.name, i.active
FROM listing_product lp
JOIN catalog_item i ON i.id = lp.item_id
WHERE lp.listing_id = $1
ORDER BY i.sort_order, i.name;

-- ListListingMachines returns the machine items of a listing with their counts.
-- name: ListListingMachines :many
SELECT i.id, i.type, i.name, i.active, lm.machine_count
FROM listing_machine lm
JOIN catalog_item i ON i.id = lm.item_id
WHERE lm.listing_id = $1
ORDER BY i.sort_order, i.name;

-- CountActiveCatalogItemsOfType counts how many of the given ids are active
-- items of the expected type. The service compares this to the number of ids it
-- sent: any shortfall means an unknown, inactive, or wrong-type id, which it
-- reports as a 422 before the type-checking trigger would raise a bare 500.
-- name: CountActiveCatalogItemsOfType :one
SELECT count(*)::bigint
FROM catalog_item
WHERE type = $1 AND active AND id = ANY($2::uuid[]);

-- ── availability calendar ──

-- InsertPeriodsUpToWeek generates every missing weekly period from the current
-- horizon forward to untilWeek in one statement. The series runs on date, not
-- timestamptz, so there is no timezone shift that could nudge a Monday to
-- Sunday. Each new period starts at the listing's weekly_capacity, zero used.
-- ON CONFLICT keeps it idempotent and safe under concurrent calls: a week that
-- already exists is left untouched (one_period_per_week). The lower bound is
-- passed in already clamped to at least the current week's Monday so past weeks
-- are never created.
-- name: InsertPeriodsUpToWeek :exec
INSERT INTO availability_period (
    id, listing_id, week_start, total_capacity, used_capacity, marked_full,
    created_at, updated_at
)
SELECT gen_random_uuid(), $1, gs::date, $4, 0, false, $5, $5
FROM generate_series($2::date, $3::date, interval '7 day') AS gs
ON CONFLICT (listing_id, week_start) DO NOTHING;

-- MaxPeriodWeek returns the furthest week_start a listing has, so a test can
-- assert horizon_until equals the last generated period.
-- name: MaxPeriodWeek :one
SELECT max(week_start)::date FROM availability_period WHERE listing_id = $1;

-- CountPeriods counts a listing's periods, backing the FR-088 "at least 13
-- periods" check.
-- name: CountPeriods :one
SELECT count(*)::bigint FROM availability_period WHERE listing_id = $1;

-- ListPeriodsInRange returns the periods of a listing within an inclusive
-- week_start range, ordered ascending, for GET /listing/me/periods. allocated
-- is the sum of active allocation quantities on each period; remaining is the
-- capacity left, floored at zero and forced to zero when the week is marked
-- full.
-- name: ListPeriodsInRange :many
SELECT
    p.week_start,
    p.total_capacity,
    p.used_capacity,
    p.marked_full,
    COALESCE((
        SELECT sum(a.quantity)::bigint
        FROM capacity_allocation a
        WHERE a.period_id = p.id AND a.reversed_at IS NULL
    ), 0) AS allocated
FROM availability_period p
WHERE p.listing_id = $1
  AND p.week_start >= $2
  AND p.week_start <= $3
ORDER BY p.week_start;

-- LockPeriodByWeek locks a single period row for an owner edit, returning its
-- current capacity and usage. A missing row means the week is beyond the
-- horizon and must be generated first. Taken only after the listing row is
-- locked, keeping the fixed lock order.
-- name: LockPeriodByWeek :one
SELECT * FROM availability_period
WHERE listing_id = $1 AND week_start = $2
FOR UPDATE;

-- UpsertPeriod writes one period's capacity and full flag on an owner edit. The
-- period row is expected to exist (EnsureHorizon generated it); ON CONFLICT
-- updates it in place, keeping used_capacity untouched so an in-flight order is
-- never disturbed.
-- name: UpsertPeriod :exec
INSERT INTO availability_period (
    id, listing_id, week_start, total_capacity, used_capacity, marked_full,
    created_at, updated_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, 0, $4, $5, $5
)
ON CONFLICT (listing_id, week_start) DO UPDATE
SET total_capacity = EXCLUDED.total_capacity,
    marked_full    = EXCLUDED.marked_full,
    updated_at     = EXCLUDED.updated_at;

-- PropagateCapacityToFuturePeriods sets total_capacity on every future period
-- that carries no active allocation (FR-089): a capacity change flows to weeks
-- the owner has not committed yet, while allocated weeks keep their agreed
-- number. The EXISTS predicate uses idx_allocation_period. week_start >= the
-- current Monday keeps past weeks frozen.
-- name: PropagateCapacityToFuturePeriods :exec
UPDATE availability_period p
SET total_capacity = $2, updated_at = $4
WHERE p.listing_id = $1
  AND p.week_start >= $3
  AND NOT EXISTS (
      SELECT 1 FROM capacity_allocation a
      WHERE a.period_id = p.id AND a.reversed_at IS NULL
  );

-- FindFutureAllocatedPeriodOverCapacity returns the earliest future period whose
-- used capacity already exceeds the proposed new weekly capacity, so a listing
-- edit that would strand a running order is rejected with a 409 that names the
-- offending week and amount rather than silently corrupting the calendar.
-- name: FindFutureAllocatedPeriodOverCapacity :one
SELECT p.week_start, p.used_capacity
FROM availability_period p
WHERE p.listing_id = $1
  AND p.week_start >= $3
  AND p.used_capacity > $2
ORDER BY p.week_start
LIMIT 1;

-- PeriodHasActiveAllocation reports whether a period carries any unreversed
-- allocation, gating the "cannot mark full" and "cannot lower below used" edits.
-- name: PeriodHasActiveAllocation :one
SELECT EXISTS (
    SELECT 1 FROM capacity_allocation a
    WHERE a.period_id = $1 AND a.reversed_at IS NULL
);
