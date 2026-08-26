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
