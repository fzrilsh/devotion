-- name: LockListingByID :one
-- Takes a row lock on a listing by its own id, so the accept path can extend the
-- calendar horizon (FR-088) under the same lock the listing owner's edits take.
SELECT * FROM capacity_listing WHERE id = $1 FOR UPDATE;

-- name: EstimateCapacityInRange :one
-- Optimistic remaining capacity across the readiness..deadline range, read BEFORE
-- the accept path takes any lock. It mirrors search's formula (search.sql,
-- FR-080/FR-088): recorded remaining over not-full periods plus an optimistic
-- estimate for weeks past horizon_until counted at weekly_capacity. A shortfall
-- here means the offer never had enough capacity, so accept fails fast with
-- INSUFFICIENT_CAPACITY without locking. If this passes but the post-lock read is
-- short, the capacity was definitively taken between the two reads
-- (CAPACITY_ALREADY_TAKEN). Deadline is rounded to Monday in Go before calling.
WITH param AS (
    SELECT
        @listing_id::uuid    AS listing_id,
        @readiness_week::date AS readiness_week,
        @deadline_week::date  AS deadline_week
)
SELECT
    coalesce(sum(pk.total_capacity - pk.used_capacity), 0)::bigint AS recorded_remaining,
    (greatest(0, (p.deadline_week - greatest(p.readiness_week, l.horizon_until + 7)) / 7 + 1) * l.weekly_capacity)::bigint AS uncreated_remaining
FROM capacity_listing l
CROSS JOIN param p
LEFT JOIN availability_period pk
       ON pk.listing_id = l.id
      AND NOT pk.marked_full
      AND pk.week_start BETWEEN p.readiness_week AND p.deadline_week
WHERE l.id = p.listing_id
GROUP BY l.id, l.weekly_capacity, l.horizon_until, p.readiness_week, p.deadline_week;

-- name: GetOfferForAccept :one
-- Loads everything the accept path needs about one offer: the offer terms, the
-- candidate and its owning listing, the request and both parties, the listing
-- capacity fields, whether the offer is the standing (latest) one, and whether
-- any candidate of the request already reached an agreement. Keyed on the offer
-- id the buyer accepts.
SELECT
    o.id                        AS offer_id,
    o.candidate_id,
    o.total_price,
    o.readiness_lead_days,
    o.proposed_by,
    o.sequence,
    (SELECT max(sequence) FROM offer WHERE candidate_id = o.candidate_id)::int AS latest_sequence,
    c.status                    AS candidate_status,
    c.subcontractor_id,
    r.id                        AS request_id,
    r.buyer_id,
    r.product_item_id,
    r.quantity,
    r.deadline,
    r.reply_due_at,
    buyer.account_id            AS buyer_account,
    sub.account_id              AS subcontractor_account,
    l.id                        AS listing_id,
    l.weekly_capacity,
    l.horizon_until,
    EXISTS (
        SELECT 1 FROM request_candidate rc
        WHERE rc.request_id = r.id AND rc.status = 'agreed'
    ) AS request_has_agreement
FROM offer o
JOIN request_candidate c    ON c.id = o.candidate_id
JOIN quota_request r        ON r.id = c.request_id
JOIN capacity_listing l     ON l.id = c.listing_id
JOIN business_profile buyer ON buyer.id = r.buyer_id
JOIN business_profile sub   ON sub.id = c.subcontractor_id
WHERE o.id = $1;

-- name: LockPeriodsInRange :many
-- Locks every candidate period in the readiness..deadline range at once, ordered
-- ascending by week_start. The order is the deadlock preventer of R-04, not tidy
-- housekeeping: two agreements touching the same periods always lock them in the
-- same order, so the second waits and then sees the reduced capacity.
SELECT * FROM availability_period
WHERE listing_id = $1 AND week_start BETWEEN $2 AND $3
ORDER BY week_start
FOR UPDATE;

-- name: InsertWorkOrder :one
-- Inserts the work order at agreement formation. readiness_week_start is stored,
-- not recomputed later (FR-084); status takes its 'accepted' default.
INSERT INTO work_order (
    candidate_id, offer_id, buyer_id, subcontractor_id,
    quantity, total_price, deadline, readiness_week_start, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: InsertAllocation :one
-- One capacity allocation row per used period (FR-077).
INSERT INTO capacity_allocation (work_order_id, period_id, quantity, created_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: RaiseUsedCapacity :one
-- Raises a period's used_capacity by the allocated amount. The
-- used_capacity_within_total CHECK is the FR-079/SC-018 storage-level safety net:
-- if the fill logic ever over-allocates, this fails hard instead of corrupting.
UPDATE availability_period
SET used_capacity = used_capacity + $2, updated_at = $3
WHERE id = $1
RETURNING *;

-- name: SetCandidateAgreed :exec
-- Marks the winning candidate agreed. The partial unique index
-- idx_one_agreement_per_request enforces at most one agreement per request, so a
-- concurrent second agreement violates it and the transaction fails (FR-036).
UPDATE request_candidate
SET status = 'agreed', updated_at = $2
WHERE id = $1;

-- name: ListOtherCandidatesToNotify :many
-- The other still-open candidates of the request, so their subcontractors learn
-- the request was agreed elsewhere. Rejected and expired candidates are left out.
SELECT c.id AS candidate_id, sub.account_id AS subcontractor_account
FROM request_candidate c
JOIN business_profile sub ON sub.id = c.subcontractor_id
WHERE c.request_id = $1 AND c.id <> $2 AND c.status NOT IN ('rejected', 'expired');

-- name: CloseOtherCandidates :exec
-- Closes the request's other candidates once one is agreed (FR-034): they move to
-- not_continued, leaving already rejected, expired, or agreed rows untouched.
UPDATE request_candidate
SET status = 'not_continued', updated_at = $3
WHERE request_id = $1 AND id <> $2 AND status NOT IN ('rejected', 'expired', 'agreed');

-- name: InsertOrderStatusHistory :exec
-- Records the opening status transition of a work order for its history trail.
INSERT INTO work_order_status_history (
    work_order_id, old_status, new_status, changed_by, by_system, note, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: LockWorkOrderForReversal :one
-- Takes a row lock on the work order whose allocation is being reversed, so the
-- reversal runs under a lock in the same spirit as formation locking the listing.
SELECT * FROM work_order WHERE id = $1 FOR UPDATE;

-- name: ListActiveAllocationsForReversal :many
-- Locks the order's still-active allocation rows together with their periods,
-- ordered ascending by week_start. The order mirrors the formation lock order of
-- R-04 (LockPeriodsInRange), the deadlock preventer: two transactions touching
-- the same periods always take them in the same order. Already reversed rows are
-- left out, so a repeat reversal refunds nothing rather than double-crediting.
SELECT a.id AS allocation_id, a.period_id, a.quantity, p.week_start
FROM capacity_allocation a
JOIN availability_period p ON p.id = a.period_id
WHERE a.work_order_id = $1 AND a.reversed_at IS NULL
ORDER BY p.week_start
FOR UPDATE OF a, p;

-- name: LowerUsedCapacity :one
-- Returns a period's used_capacity to its pre-order value by subtracting the
-- reversed amount. The inverse of RaiseUsedCapacity; the quantity comes from the
-- allocation row being reversed, so used_capacity never drops below zero.
UPDATE availability_period
SET used_capacity = used_capacity - $2, updated_at = $3
WHERE id = $1
RETURNING *;

-- name: MarkAllocationReversed :exec
-- Marks one allocation row reversed without deleting it (FR-020), keeping the
-- audit trail. The reversed_at guard makes the write idempotent under a repeat.
UPDATE capacity_allocation
SET reversed_at = $2
WHERE id = $1 AND reversed_at IS NULL;
