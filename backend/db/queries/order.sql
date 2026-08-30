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

-- name: GetWorkOrderForView :one
-- Loads one work order with the fields WorkOrderDetail needs beyond the row
-- itself: both parties' account ids (for the party guard), the request's product
-- item, and the offer's readiness lead. Keyed on the work order id; the caller
-- checks the account ids against the principal so a non-party sees a 404.
-- has_open_dispute carries whether any unresolved dispute exists on the order, so
-- the read layer feeds it to order.IsAutoConfirmDue: an open dispute halts the
-- lazy auto-confirm exactly as the NOT EXISTS guard halts the ticker, so the two
-- layers never disagree on a disputed order (FR-070).
SELECT
    wo.id,
    wo.candidate_id,
    wo.offer_id,
    wo.buyer_id,
    wo.subcontractor_id,
    wo.quantity,
    wo.total_price,
    wo.deadline,
    wo.readiness_week_start,
    wo.status,
    wo.shipped_at,
    wo.auto_confirm_base_at,
    wo.confirmed_at,
    wo.auto_confirmed,
    wo.cancelled_by_id,
    wo.cancellation_reason,
    wo.cancelled_at,
    wo.created_at,
    buyer.account_id AS buyer_account,
    sub.account_id   AS subcontractor_account,
    r.product_item_id,
    o.readiness_lead_days,
    EXISTS (
        SELECT 1 FROM dispute d
        WHERE d.work_order_id = wo.id AND d.status <> 'resolved'
    ) AS has_open_dispute
FROM work_order wo
JOIN business_profile buyer ON buyer.id = wo.buyer_id
JOIN business_profile sub   ON sub.id = wo.subcontractor_id
JOIN request_candidate c    ON c.id = wo.candidate_id
JOIN quota_request r        ON r.id = c.request_id
JOIN offer o                ON o.id = wo.offer_id
WHERE wo.id = $1;

-- name: GetWorkOrderContacts :one
-- Loads both parties' contact details for one work order (FR-092): each side's
-- business name, email, and WhatsApp number, plus the account ids the handler
-- needs for the party guard. The handler compares the caller's account id to
-- buyer_account and subcontractor_account and returns only the counterparty's
-- block, so a non-party (or a missing order) collapses to a 404 and the caller
-- never sees their own side echoed back. Keyed on the work order id.
SELECT
    wo.id,
    buyer.account_id            AS buyer_account,
    buyer.business_name         AS buyer_business_name,
    ba.email                    AS buyer_email,
    ba.phone                    AS buyer_phone,
    sub.account_id              AS subcontractor_account,
    sub.business_name           AS subcontractor_business_name,
    sa.email                    AS subcontractor_email,
    sa.phone                    AS subcontractor_phone
FROM work_order wo
JOIN business_profile buyer ON buyer.id = wo.buyer_id
JOIN business_profile sub   ON sub.id = wo.subcontractor_id
JOIN user_account ba        ON ba.id = buyer.account_id
JOIN user_account sa        ON sa.id = sub.account_id
WHERE wo.id = $1;

-- name: LockWorkOrderForStatusChange :one
-- Row-locks a work order before a status transition so a concurrent status
-- change or cancellation on the same order serializes, matching the reversal
-- lock spirit (R-04).
SELECT * FROM work_order WHERE id = $1 FOR UPDATE;

-- name: ListWorkOrderStatusHistory :many
-- The status trail of one work order, oldest first, for WorkOrderDetail. Rides
-- idx_status_history_order (work_order_id, created_at) so it stays ordered
-- without a sort.
SELECT new_status, created_at, note
FROM work_order_status_history
WHERE work_order_id = $1
ORDER BY created_at, id;

-- name: ListWorkOrderAllocations :many
-- The still-active allocation periods of one work order with the period figures
-- WorkOrderDetail renders, ordered ascending by week_start. Reversed rows are
-- left out so a cancelled order shows no live allocation.
SELECT p.week_start, p.total_capacity, p.used_capacity, p.marked_full
FROM capacity_allocation a
JOIN availability_period p ON p.id = a.period_id
WHERE a.work_order_id = $1 AND a.reversed_at IS NULL
ORDER BY p.week_start;

-- name: UpdateWorkOrderStatus :one
-- Advances a work order to its next forward status. shipped_at is stamped only
-- on the move into 'shipped' and left untouched otherwise, so it records the
-- moment shipment was declared (the auto-confirm clock start, FR-068).
UPDATE work_order
SET status = $2,
    shipped_at = CASE WHEN $2 = 'shipped'::work_order_status THEN $3 ELSE shipped_at END
WHERE id = $1
RETURNING *;

-- name: CancelWorkOrder :one
-- Records a pre-production self-cancellation on the order itself (FR-065): the
-- cancelling party's profile id, the reason, and the moment, moving status to
-- 'cancelled'. Together the four columns satisfy the cancellation_complete
-- CHECK. The allocation reversal (FR-020) runs separately under the same tx.
UPDATE work_order
SET status = 'cancelled',
    cancelled_by_id = $2,
    cancellation_reason = $3,
    cancelled_at = $4
WHERE id = $1
RETURNING *;

-- name: ListWorkOrdersForParty :many
-- One party's work orders newest first, keyset paginated on (created_at, id) so
-- the order is stable across pages (FR-038). role_filter selects the side:
-- 'as_buyer' matches the buyer profile, 'as_subcontractor' the subcontractor
-- profile, any other value matches either. An empty status_filter array means no
-- status restriction; otherwise only the listed statuses pass. The filter is a
-- text[] cast against wo.status::text, not a work_order_status[]: pgx cannot encode
-- a slice of the named enum type without the enum OID registered on the pool, so an
-- enum-array parameter fails to encode on every request (empty slice included). The
-- text cast sidesteps that with no per-connection type registration (Principle IV).
-- A null before_created is the first page. has_open_dispute rides along so
-- listItemView feeds it to order.IsAutoConfirmDue, keeping the list's lazy
-- auto-confirm in step with the detail view and the ticker on a disputed order
-- (FR-070).
SELECT wo.*,
    EXISTS (
        SELECT 1 FROM dispute d
        WHERE d.work_order_id = wo.id AND d.status <> 'resolved'
    ) AS has_open_dispute
FROM work_order wo
WHERE (
        (sqlc.arg(role_filter)::text = 'as_buyer' AND wo.buyer_id = sqlc.arg(profile_id)::uuid)
        OR (sqlc.arg(role_filter)::text = 'as_subcontractor' AND wo.subcontractor_id = sqlc.arg(profile_id)::uuid)
        OR (sqlc.arg(role_filter)::text NOT IN ('as_buyer', 'as_subcontractor')
            AND (wo.buyer_id = sqlc.arg(profile_id)::uuid OR wo.subcontractor_id = sqlc.arg(profile_id)::uuid))
    )
  AND (
        cardinality(sqlc.arg(status_filter)::text[]) = 0
        OR wo.status::text = ANY(sqlc.arg(status_filter)::text[])
    )
  AND (
        sqlc.narg(before_created)::timestamptz IS NULL
        OR (wo.created_at, wo.id) < (sqlc.narg(before_created)::timestamptz, sqlc.narg(before_id)::uuid)
    )
ORDER BY wo.created_at DESC, wo.id DESC
LIMIT sqlc.arg(page_limit)::int;

-- name: ListShippedDueForAutoConfirm :many
-- The shipped orders whose 7-day auto-confirm instant has arrived (FR-068),
-- for the in-process ticker to close. The due test is on the effective base
-- COALESCE(auto_confirm_base_at, shipped_at) <= $1 with $1 = now -
-- AutoConfirmWindow computed by the caller from the injected Clock, so the
-- boundary matches order.AutoConfirmBase + order.IsAutoConfirmDue exactly and no
-- wall clock is read in SQL. Normally the base is shipped_at; on an order whose
-- dispute closed "continued" after shipment it is the mediation-close instant, so
-- the clock restarts from mediation, not from the original shipment (FR-070,
-- data-model.md). Rides idx_order_auto_confirm on the same COALESCE expression
-- WHERE status='shipped'. An order with an open dispute is excluded by the NOT
-- EXISTS guard, so reporting a dispute stops the auto-confirm count without moving
-- the order off 'shipped' (FR-070): the dispute row, not the work-order status, is
-- what halts the clock, and admin moves the order to in_mediation separately
-- (T071). Returns each order's id and both parties' account ids for the closure
-- notice.
SELECT wo.id, buyer.account_id AS buyer_account, sub.account_id AS subcontractor_account
FROM work_order wo
JOIN business_profile buyer ON buyer.id = wo.buyer_id
JOIN business_profile sub   ON sub.id = wo.subcontractor_id
WHERE wo.status = 'shipped'
  AND COALESCE(wo.auto_confirm_base_at, wo.shipped_at) <= sqlc.arg(due_before)::timestamptz
  AND NOT EXISTS (
        SELECT 1 FROM dispute d
        WHERE d.work_order_id = wo.id AND d.status <> 'resolved'
  )
ORDER BY COALESCE(wo.auto_confirm_base_at, wo.shipped_at);

-- name: AutoConfirmWorkOrder :one
-- Closes one shipped order as system-confirmed (FR-068): status to 'confirmed',
-- auto_confirmed true (marking this closure as the system's, not a party's), and
-- confirmed_at stamped from the caller's Clock instant. The status = 'shipped'
-- guard makes the write a no-op if a party confirmed the order since the scan. The
-- NOT EXISTS open-dispute guard closes the race where a dispute is reported between
-- the scan and this update: a disputed order stays open even though its status is
-- still 'shipped' (FR-070). A returned row means this call did the closing.
-- Together the columns satisfy the auto_confirm_needs_confirmation and
-- shipped_before_confirmed CHECKs.
UPDATE work_order
SET status = 'confirmed', auto_confirmed = true, confirmed_at = $2
WHERE work_order.id = $1 AND work_order.status = 'shipped'
  AND NOT EXISTS (
        SELECT 1 FROM dispute d
        WHERE d.work_order_id = work_order.id AND d.status <> 'resolved'
  )
RETURNING *;

-- name: PartyConfirmWorkOrder :one
-- Closes one shipped order as buyer-confirmed (FR-047, FR-068): status to
-- 'confirmed', auto_confirmed false (this is the buyer's manual acceptance, not
-- the system's 7-day closure, so the two are distinguishable in the trail), and
-- confirmed_at stamped from the caller's Clock instant. The status = 'shipped'
-- guard makes the write a no-op if the order left 'shipped' since the caller read
-- it (e.g. the ticker already auto-confirmed, or a dispute moved it to mediation),
-- so a returned row means this call did the closing. The NOT EXISTS open-dispute
-- guard mirrors AutoConfirmWorkOrder: an order with an unresolved dispute stays
-- open even while its status is still 'shipped' (FR-070). confirmed_at >= shipped_at
-- holds by the shipped_before_confirmed CHECK since the order had shipped.
UPDATE work_order
SET status = 'confirmed', auto_confirmed = false, confirmed_at = $2
WHERE work_order.id = $1 AND work_order.status = 'shipped'
  AND NOT EXISTS (
        SELECT 1 FROM dispute d
        WHERE d.work_order_id = work_order.id AND d.status <> 'resolved'
  )
RETURNING *;

-- name: ListShippedApproachingAutoConfirm :many
-- The shipped orders inside the FR-069 warning lead that have not yet been warned:
-- the effective base COALESCE(auto_confirm_base_at, shipped_at) is within
-- (warn_after, due_after] so the auto-confirm instant is between AutoConfirmWarnLead
-- and now, and confirm_warn_sent_at IS NULL dedups so the ticker warns each order
-- once, not on every tick. The caller passes warn_after = now - AutoConfirmWindow +
-- AutoConfirmWarnLead and due_after = now - AutoConfirmWindow, both from the Clock,
-- so the window matches order.IsAutoConfirmApproaching over the same effective base.
-- Normally the base is shipped_at; on an order whose dispute closed "continued"
-- after shipment it is the mediation-close instant, so the warning tracks the
-- restarted clock (FR-070). An order with an open dispute is excluded by the NOT
-- EXISTS guard, so a disputed order is neither warned about nor auto-closed
-- (FR-070). Rides idx_order_auto_confirm on the same COALESCE expression. Returns
-- the id, the buyer account (the only party FR-069 warns), and the effective base
-- so the caller can name the auto-confirm date in the notice.
SELECT wo.id, buyer.account_id AS buyer_account,
    COALESCE(wo.auto_confirm_base_at, wo.shipped_at)::timestamptz AS auto_confirm_base
FROM work_order wo
JOIN business_profile buyer ON buyer.id = wo.buyer_id
WHERE wo.status = 'shipped'
  AND wo.confirm_warn_sent_at IS NULL
  AND COALESCE(wo.auto_confirm_base_at, wo.shipped_at) > sqlc.arg(due_before)::timestamptz
  AND COALESCE(wo.auto_confirm_base_at, wo.shipped_at) <= sqlc.arg(warn_before)::timestamptz
  AND NOT EXISTS (
        SELECT 1 FROM dispute d
        WHERE d.work_order_id = wo.id AND d.status <> 'resolved'
  )
ORDER BY COALESCE(wo.auto_confirm_base_at, wo.shipped_at);

-- name: MarkConfirmWarnSent :exec
-- Stamps confirm_warn_sent_at so the FR-069 approaching notice is sent to the
-- buyer once. The IS NULL guard keeps it idempotent if two overlapping ticker
-- instances both scanned before either stamped (the advisory lock makes that
-- rare, but the guard removes the race entirely).
UPDATE work_order
SET confirm_warn_sent_at = $2
WHERE id = $1 AND confirm_warn_sent_at IS NULL;

-- name: ListLateWorkOrdersForAdmin :many
-- The active orders whose delivery deadline has passed, newest first, keyset
-- paginated on (created_at, id) for the admin monitoring list (FR-045). The
-- status set is exactly idx_order_deadline_active's predicate, so the query rides
-- that partial index instead of adding a new one: an order counts as late only
-- while it is still open (accepted, production, completed, shipped); a confirmed
-- or cancelled order is out even if its deadline has long passed, and an
-- in-mediation order is excluded because a dispute already has admin attention.
-- deadline < before_cutoff::date is the past-deadline test with before_cutoff =
-- order.PastDeadlineCutoff(now), the start of now's WIB day, computed by the
-- caller from the injected Clock so the boundary matches order.IsPastDeadline and
-- no wall clock is read in SQL (Rule 5). has_open_dispute rides along so the admin
-- render reuses listItemView with the same lazy auto-confirm as the party list.
SELECT wo.*,
    EXISTS (
        SELECT 1 FROM dispute d
        WHERE d.work_order_id = wo.id AND d.status <> 'resolved'
    ) AS has_open_dispute
FROM work_order wo
WHERE wo.status IN ('accepted', 'production', 'completed', 'shipped')
  AND wo.deadline < sqlc.arg(before_cutoff)::date
  AND (
        sqlc.narg(before_created)::timestamptz IS NULL
        OR (wo.created_at, wo.id) < (sqlc.narg(before_created)::timestamptz, sqlc.narg(before_id)::uuid)
    )
ORDER BY wo.created_at DESC, wo.id DESC
LIMIT sqlc.arg(page_limit)::int;

-- name: ListLateWorkOrdersToNotify :many
-- The active orders past their deadline that have not yet had the late-delivery
-- notice sent, for the in-process ticker to notify (FR-045). It mirrors
-- ListLateWorkOrdersForAdmin's status set and deadline test (both share
-- order.PastDeadlineCutoff), adding late_notified_at IS NULL so each late order is
-- notified once, not on every tick. Returns each order's id and both parties'
-- account ids, since FR-045 alerts both sides. Rides idx_order_deadline_active.
SELECT wo.id, buyer.account_id AS buyer_account, sub.account_id AS subcontractor_account
FROM work_order wo
JOIN business_profile buyer ON buyer.id = wo.buyer_id
JOIN business_profile sub   ON sub.id = wo.subcontractor_id
WHERE wo.status IN ('accepted', 'production', 'completed', 'shipped')
  AND wo.deadline < sqlc.arg(before_cutoff)::date
  AND wo.late_notified_at IS NULL
ORDER BY wo.deadline;

-- name: MarkLateNotified :exec
-- Stamps late_notified_at so the FR-045 late-delivery notice is sent once per
-- order. The IS NULL guard keeps it idempotent if two overlapping ticker
-- instances both scanned before either stamped (the advisory lock makes that
-- rare, but the guard removes the race entirely). The stamp is never cleared,
-- even when mediation later closes, so a resolved late order is not re-notified.
UPDATE work_order
SET late_notified_at = $2
WHERE id = $1 AND late_notified_at IS NULL;

-- name: InsertPaymentRecord :one
-- Records one party's payment statement on a work order (FR-041). No money amount
-- is stored: the platform neither holds nor verifies funds (FR-040, FR-042), so
-- the row carries only the direction (sent/received), the date the party states,
-- an optional free note, and the Clock-stamped created_at. The
-- one_statement_per_party_per_direction unique constraint means a party can state
-- each direction at most once; a repeat violates it and the caller turns the
-- 23505 into a readable PAYMENT_STATEMENT_EXISTS 409.
INSERT INTO payment_record (
    work_order_id, profile_id, direction, date, note, created_at
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListPaymentRecordsForWorkOrder :many
-- Every payment statement on one work order for WorkOrderDetail (FR-041). Ordered
-- by created_at so the two parties' statements read in the order they were made;
-- both directions from both parties are visible, so a disagreement (one party
-- states sent, the counterparty never states received) is apparent to the parties
-- and to admin when a dispute is reported (FR-043).
SELECT id, work_order_id, profile_id, direction, date, note, created_at
FROM payment_record
WHERE work_order_id = $1
ORDER BY created_at, id;

-- name: InsertDispute :one
-- Opens a dispute on a work order (FR-046). status takes its 'reported' default.
-- The idx_one_open_dispute partial unique index (work_order_id WHERE status <>
-- 'resolved') means at most one dispute may be open per order; a second report
-- violates it and the caller turns the 23505 into a readable DISPUTE_ALREADY_OPEN
-- 409 (FR-046). reporter_id is the reporting party's business profile id.
INSERT INTO dispute (
    work_order_id, reporter_id, report_body, created_at
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: MoveWorkOrderToMediation :one
-- Moves a shipped or otherwise-active order into 'in_mediation' when a dispute is
-- reported (FR-070). The status = 'shipped' scan of ListShippedDueForAutoConfirm
-- excludes 'in_mediation', so opening a dispute before the 7-day deadline stops
-- the auto-confirm count. confirm_warn_sent_at is deliberately left untouched: a
-- buyer already warned stays warned even if the dispute later closes without
-- cancelling (data-model.md). The status guard rejects moving a terminal order
-- (confirmed/cancelled), so a returned row means this call did the move.
UPDATE work_order
SET status = 'in_mediation'
WHERE id = $1 AND status IN ('accepted', 'production', 'completed', 'shipped')
RETURNING *;

-- name: MarkDisputeInMediation :exec
-- Marks a reported dispute as in_mediation when the admin takes up the case
-- (FR-046). Runs in the same transaction as MoveWorkOrderToMediation so an order
-- in mediation always has its dispute in mediation. The status = 'reported' guard
-- keeps this from re-opening a dispute the admin already resolved; a resolved
-- dispute is rejected earlier by the handler's GetDisputeForAdmin status check, so
-- in practice this only ever advances 'reported' -> 'in_mediation'.
UPDATE dispute
SET status = 'in_mediation'
WHERE id = $1 AND status = 'reported';

-- name: ListDisputesForAdmin :many
-- The admin dispute queue, newest first, keyset paginated on (created_at, id) so
-- the order is stable across pages (FR-046). An optional status_filter narrows to
-- one DisputeStatus (reported / in_mediation / resolved); a null filter returns
-- every dispute regardless of status. A null before_created is the first page.
-- The unresolved slice of this list rides idx_dispute_queue(created_at); a status
-- filter of 'resolved' scans without that partial index, which is acceptable for
-- an admin-only review list. Returns the full dispute row so the handler renders
-- the contract Dispute shape (reporter_id -> reporter_profile_id, liable_party_id
-- -> liable_profile_id) without a second read.
SELECT * FROM dispute
WHERE (
        sqlc.narg(status_filter)::text IS NULL
        OR status::text = sqlc.narg(status_filter)::text
    )
  AND (
        sqlc.narg(before_created)::timestamptz IS NULL
        OR (created_at, id) < (sqlc.narg(before_created)::timestamptz, sqlc.narg(before_id)::uuid)
    )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit)::int;

-- name: GetDisputeForAdmin :one
-- Loads and row-locks one dispute for the mediate and resolve paths (FR-046,
-- FR-067). FOR UPDATE serializes two admins acting on the same dispute: the second
-- waits, then sees the resolved status and is rejected by the resolve guard. Keyed
-- on the dispute id from the path; a missing row is the handler's 404.
SELECT * FROM dispute WHERE id = $1 FOR UPDATE;

-- name: ResolveDispute :one
-- Records the admin's mediation decision and closes the dispute (FR-067, FR-072).
-- status moves to 'resolved'; result is the explicit outcome (cancelled /
-- continued / confirmed) the resolution_complete CHECK now requires; admin_note
-- carries the mandated note; allocation_reversed and liable_party_id record
-- whether capacity was returned and which party bears the cancellation in the
-- completion rate (FR-072); handled_by is the admin account and resolved_at the
-- Clock instant. The status <> 'resolved' guard makes a repeat resolve a no-op so
-- two admins cannot close the same dispute twice. The work-order side effects
-- (cancel + reversal, restore, or force-confirm) run in the same transaction.
UPDATE dispute
SET status = 'resolved',
    result = $2,
    allocation_reversed = $3,
    liable_party_id = $4,
    admin_note = $5,
    handled_by = $6,
    resolved_at = $7
WHERE id = $1 AND status <> 'resolved'
RETURNING *;

-- name: GetStatusBeforeMediation :one
-- The status a work order held immediately before it entered mediation, read from
-- the old_status of the most recent transition into 'in_mediation' (FR-067). The
-- 'continued' resolution restores the order to this status. old_status is read
-- straight off the mediation history row rather than inferred, so a restore lands
-- on exactly the status the order left, even if it moved through several statuses
-- earlier in its life. Newest such row wins in case an order was mediated more
-- than once over its lifetime.
SELECT old_status
FROM work_order_status_history
WHERE work_order_id = $1 AND new_status = 'in_mediation'
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: RestoreWorkOrderStatus :one
-- Returns a mediated order to a prior status when a dispute closes 'continued'
-- (FR-067): the order rejoins its production chain at the status it left. The
-- status = 'in_mediation' guard means only an order actually in mediation is
-- restored, so a returned row confirms this call did the move. auto_confirm_base_at
-- and confirm_warn_sent_at are left untouched here; RestartAutoConfirmClock handles
-- them separately, and only when the restored status is 'shipped'.
UPDATE work_order
SET status = $2
WHERE id = $1 AND status = 'in_mediation'
RETURNING *;

-- name: RestartAutoConfirmClock :exec
-- Restarts the 7-day auto-confirm clock from the mediation-close instant on an
-- order restored to 'shipped' by a 'continued' resolution (FR-068, data-model.md).
-- auto_confirm_base_at becomes the mediation-close time, so AutoConfirmBase reads
-- it instead of shipped_at and the window counts from mediation, not the original
-- shipment; shipped_at is left as the historical shipment fact. confirm_warn_sent_at
-- is reset to NULL so the buyer is warned afresh before the new deadline. Called
-- only on the continued+shipped branch, where the restored status is 'shipped' so
-- the auto_confirm_base_after_shipped CHECK (base >= shipped_at) holds.
UPDATE work_order
SET auto_confirm_base_at = $2, confirm_warn_sent_at = NULL
WHERE id = $1;

-- name: ForceConfirmWorkOrder :one
-- Forces a mediated order to 'confirmed' when the admin closes the dispute that way
-- (FR-067): the admin accepts the order on the buyer's behalf. auto_confirmed stays
-- false because this is an admin decision, not the system's 7-day closure, so the
-- two are distinguishable in the trail; confirmed_at is the Clock instant. The
-- status = 'in_mediation' guard means a returned row confirms this call did the
-- move. confirmed_at >= shipped_at holds by shipped_before_confirmed when the order
-- had shipped; an order confirmed from an earlier status has shipped_at NULL and the
-- CHECK is satisfied vacuously.
UPDATE work_order
SET status = 'confirmed', auto_confirmed = false, confirmed_at = $2
WHERE id = $1 AND status = 'in_mediation'
RETURNING *;
