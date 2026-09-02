-- name: InsertReview :one
-- Records one party's review of the other on a finished work order (FR-047).
-- rating is 1..5 (enforced by the rating_one_to_five CHECK), text is optional.
-- The one_review_per_order_per_reviewer unique constraint means each party may
-- review an order once; a repeat violates it and the caller turns the 23505 into
-- a readable REVIEW_ALREADY_SUBMITTED 409 (FR-049). Reviews are never anonymous:
-- reviewer_id is stored and shown (FR-050). hidden takes its false default;
-- admin hiding is a later moderation action (T069), not a field the author sets.
INSERT INTO review (
    work_order_id, reviewer_id, reviewee_id, rating, text, created_at
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ReviewExistsForOrderReviewer :one
-- Reports whether a given profile has already reviewed a given work order, the
-- not-yet-reviewed leg of WorkOrderDetail.can_review (FR-047). The detail view
-- offers the review button only while this is false, so the client never shows
-- an action the one_review_per_order_per_reviewer constraint would reject. Rides
-- that same unique key (work_order_id, reviewer_id).
SELECT EXISTS (
    SELECT 1 FROM review
    WHERE work_order_id = $1 AND reviewer_id = $2
);

-- name: GetReviewForResponse :one
-- Reloads one review with the author's business name and the transaction date,
-- so the 201 body carries the same shape the public list returns. Keyed on the
-- review id just inserted.
SELECT
    r.id,
    r.work_order_id,
    r.reviewer_id,
    r.reviewee_id,
    r.rating,
    r.text,
    r.hidden,
    r.created_at,
    author.business_name AS author_business_name,
    coalesce(wo.confirmed_at, wo.created_at)::date AS transaction_date
FROM review r
JOIN business_profile author ON author.id = r.reviewer_id
JOIN work_order wo           ON wo.id = r.work_order_id
WHERE r.id = $1;

-- name: LockReviewForHide :one
-- Locks one review row FOR UPDATE so the hide decision reads its current hidden
-- state and writes the new one without a competing admin racing between the two
-- (mirrors the lock-then-decide pattern the other admin moderation paths use).
-- Returns just enough to decide: the id and whether it is already hidden.
SELECT id, hidden
FROM review
WHERE id = $1
FOR UPDATE;

-- name: HideReview :exec
-- Marks one review hidden with the admin's identity, the moment, and the reason
-- (FR-050). The hiding_complete CHECK enforces that all three accompany hidden;
-- the handler fills them and rejects an empty reason first, so a caller sees a
-- readable validation error rather than a raw constraint violation. Setting
-- hidden true is the whole action: the average and the public list both filter
-- NOT hidden already, so the row leaves both at once with no second rule.
UPDATE review
SET hidden = true,
    hidden_by = $2,
    hidden_at = $3,
    hidden_reason = $4
WHERE id = $1;

-- name: ListReviewsForProfile :many
-- One profile's received reviews, newest first, keyset paginated on
-- (created_at, id) so the order is stable across pages (FR-048). Hidden reviews
-- are excluded: a review admin took down must not reappear in the public list,
-- and the same NOT hidden filter is what SearchReputation applies to the average,
-- so the list and the average never disagree (FR-050). Rides
-- idx_review_reviewee (reviewee_id) WHERE NOT hidden. A null before_created is
-- the first page. The author's business name rides along because reviews are not
-- anonymous, and transaction_date names the order the review is about.
SELECT
    r.id,
    r.work_order_id,
    r.reviewer_id,
    r.reviewee_id,
    r.rating,
    r.text,
    r.hidden,
    r.created_at,
    author.business_name AS author_business_name,
    coalesce(wo.confirmed_at, wo.created_at)::date AS transaction_date
FROM review r
JOIN business_profile author ON author.id = r.reviewer_id
JOIN work_order wo           ON wo.id = r.work_order_id
WHERE r.reviewee_id = sqlc.arg(reviewee_id)::uuid
  AND NOT r.hidden
  AND (
        sqlc.narg(before_created)::timestamptz IS NULL
        OR (r.created_at, r.id) < (sqlc.narg(before_created)::timestamptz, sqlc.narg(before_id)::uuid)
    )
ORDER BY r.created_at DESC, r.id DESC
LIMIT sqlc.arg(page_limit)::int;
