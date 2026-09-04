-- InsertQuotaRequest creates one quota request. Both created_at and reply_due_at
-- are supplied by the caller from the injected Clock (FR-082, Rule 5): the table
-- has no DEFAULT now(), and its only time constraint enforces reply_due_at >
-- created_at. The 72-hour window is computed in the service, not here.
-- name: InsertQuotaRequest :one
INSERT INTO quota_request (buyer_id, product_item_id, quantity, material, deadline, note, reply_due_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- InsertRequestCandidate links one listing to a request as a candidate. status
-- defaults to 'awaiting_reply'. The trg_reject_self_request trigger raises when
-- subcontractor_id equals the request's buyer_id; the service rejects that case
-- first (FR-083), so this insert is the safety net.
-- name: InsertRequestCandidate :one
INSERT INTO request_candidate (request_id, listing_id, subcontractor_id, updated_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- GetCandidateListings resolves each listing id to its owning profile and that
-- profile's account, and reports whether the listing is published. The service
-- uses this to reject unknown or unpublished listings (422) and to detect a
-- listing owned by the searcher's own profile (FR-083) before any insert.
-- name: GetCandidateListings :many
SELECT l.id AS listing_id, l.profile_id, l.published, p.account_id, p.business_name
FROM capacity_listing l
JOIN business_profile p ON p.id = l.profile_id
WHERE l.id = ANY($1::uuid[]);

-- ListQuotaRequestsByBuyer returns one keyset page of a buyer's own requests,
-- newest first, ordered by (created_at, id) so the order is total and stable
-- across pages (FR-030, FR-080). The cursor tuple admits every row on the first
-- page via sentinels above the maxima.
-- name: ListQuotaRequestsByBuyer :many
SELECT *
FROM quota_request
WHERE buyer_id = $1
  AND (created_at, id) < ($2::timestamptz, $3::uuid)
ORDER BY created_at DESC, id DESC
LIMIT $4;

-- ListCandidatesByRequests returns every candidate of the given requests, joined
-- to its subcontractor profile for the business name, ordered so a request's
-- candidates group together deterministically.
-- name: ListCandidatesByRequests :many
SELECT c.id AS candidate_id, c.request_id, c.listing_id, c.subcontractor_id,
       c.status, c.rejection_reason, p.business_name
FROM request_candidate c
JOIN business_profile p ON p.id = c.subcontractor_id
WHERE c.request_id = ANY($1::uuid[])
ORDER BY c.request_id, c.id;

-- GetCandidateForOffer loads one candidate with everything the offer and reject
-- paths need in a single read: the candidate's status and its owning
-- subcontractor account (for the role check), the request's quantity, deadline,
-- and reply window (FR-035, FR-090, expiry), and the listing's capacity shape
-- (weekly_capacity, readiness_lead_days, horizon_until) plus the buyer account to
-- notify. It is the incoming-side counterpart to GetCandidateListings.
-- name: GetCandidateForOffer :one
SELECT
    c.id                  AS candidate_id,
    c.status              AS candidate_status,
    c.subcontractor_id,
    sub.account_id        AS subcontractor_account,
    r.id                  AS request_id,
    r.quantity,
    r.deadline,
    r.reply_due_at,
    r.buyer_id,
    buyer.account_id      AS buyer_account,
    l.id                  AS listing_id,
    l.weekly_capacity,
    l.readiness_lead_days,
    l.horizon_until
FROM request_candidate c
JOIN quota_request r      ON r.id = c.request_id
JOIN capacity_listing l   ON l.id = c.listing_id
JOIN business_profile sub ON sub.id = c.subcontractor_id
JOIN business_profile buyer ON buyer.id = r.buyer_id
WHERE c.id = $1;

-- RemainingCapacityForOffer sums the subcontractor's remaining capacity across
-- the readiness..deadline week range for FR-035, mirroring the capacity CTE of
-- SearchCandidates: recorded remaining over non-full availability periods in
-- range, plus periods past horizon_until counted optimistically as full weekly
-- capacity (FR-088). The caller computes readiness_week and deadline_week in Go
-- so week rounding stays in one place (Rule 4).
-- name: RemainingCapacityForOffer :one
WITH param AS (
    SELECT
        @listing_id::uuid       AS listing_id,
        @readiness_week::date    AS readiness_week,
        @deadline_week::date     AS deadline_week,
        @weekly_capacity::int    AS weekly_capacity,
        @horizon_until::date     AS horizon_until
)
SELECT (
    coalesce(sum(pk.total_capacity - pk.used_capacity), 0)
    + greatest(0, (
        (p.deadline_week - greatest(p.readiness_week, p.horizon_until + 7)) / 7 + 1
    )) * p.weekly_capacity
)::bigint AS remaining_capacity
FROM param p
LEFT JOIN availability_period pk
       ON pk.listing_id = p.listing_id
      AND NOT pk.marked_full
      AND pk.week_start BETWEEN p.readiness_week AND p.deadline_week
GROUP BY p.deadline_week, p.readiness_week, p.horizon_until, p.weekly_capacity;

-- InsertOffer appends one offer to a candidate's chain, computing the next
-- sequence as max+1 so every counter-offer is a new row and the full history is
-- preserved (FR-033). created_at comes from the injected Clock (Rule 5).
-- name: InsertOffer :one
INSERT INTO offer (candidate_id, sequence, proposed_by, total_price, readiness_lead_days, note, created_at)
VALUES (
    $1,
    coalesce((SELECT max(sequence) FROM offer WHERE candidate_id = $1), 0) + 1,
    $2, $3, $4, $5, $6
)
RETURNING *;

-- GetOfferForCounter loads one offer with the context the counter-offer path
-- needs: the candidate it belongs to, that candidate's status, and both parties'
-- accounts so the service can check the caller alternates with the last proposer
-- and notify the other side (FR-033).
-- name: GetOfferForCounter :one
SELECT
    o.id            AS offer_id,
    o.candidate_id,
    o.proposed_by,
    o.sequence,
    o.readiness_lead_days,
    c.status        AS candidate_status,
    r.id            AS request_id,
    sub.account_id  AS subcontractor_account,
    buyer.account_id AS buyer_account
FROM offer o
JOIN request_candidate c  ON c.id = o.candidate_id
JOIN quota_request r      ON r.id = c.request_id
JOIN business_profile sub ON sub.id = c.subcontractor_id
JOIN business_profile buyer ON buyer.id = r.buyer_id
WHERE o.id = $1
ORDER BY o.sequence DESC
LIMIT 1;

-- SetCandidateStatus moves a candidate to a new status (offered on a reply,
-- FR-031), stamping updated_at from the Clock.
-- name: SetCandidateStatus :exec
UPDATE request_candidate
SET status = $2, updated_at = $3
WHERE id = $1;

-- RejectCandidate marks a candidate rejected with the subcontractor's reason
-- (FR-031), stamping updated_at from the Clock.
-- name: RejectCandidate :exec
UPDATE request_candidate
SET status = 'rejected', rejection_reason = $2, updated_at = $3
WHERE id = $1;

-- ListCandidatesToExpire returns candidates still awaiting a reply whose
-- request's 72-hour window has lapsed, for the in-process ticker to expire and
-- notify the buyer (FR-037). The before_cutoff bound is the current instant,
-- passed from the injected Clock (Rule 5), matching order.IsRequestExpired's
-- inclusive boundary via <=. Each row carries the candidate id (to expire), the
-- request id (for the buyer's deep link), and the buyer account (to notify),
-- ordered by request so a request's lapsed candidates group together.
-- name: ListCandidatesToExpire :many
SELECT c.id AS candidate_id, r.id AS request_id, buyer.account_id AS buyer_account
FROM request_candidate c
JOIN quota_request r        ON r.id = c.request_id
JOIN business_profile buyer ON buyer.id = r.buyer_id
WHERE c.status = 'awaiting_reply'
  AND r.reply_due_at <= sqlc.arg(before_cutoff)::timestamptz
ORDER BY r.id, c.id;

-- ExpireCandidate moves an unanswered candidate to 'expired' once its reply
-- window has lapsed (FR-037). The status = 'awaiting_reply' guard makes it a
-- no-op if the subcontractor replied (offered/rejected) or a race already
-- expired it between the scan and this update, so two overlapping ticker
-- instances expire a candidate once. It reports the rows affected so the caller
-- notifies the buyer only for a candidate this pass actually expired.
-- name: ExpireCandidate :execrows
UPDATE request_candidate
SET status = 'expired', updated_at = $2
WHERE id = $1 AND status = 'awaiting_reply';

-- RequestHasStandingOffer reports whether a request still has a candidate that
-- replied with an offer or was agreed, so the expiry job tells the buyer the
-- request lapsed "tanpa penawaran" only when none did (AS-7, FR-037). A rejected
-- or not-continued candidate is not a standing offer, matching the notice body.
-- name: RequestHasStandingOffer :one
SELECT EXISTS (
    SELECT 1 FROM request_candidate
    WHERE request_id = $1 AND status IN ('offered', 'agreed')
);

-- GetRequestForBuyer loads one request owned by a buyer account, for the detail
-- view (FR-032). The buyer account guard makes a request that is not the
-- caller's a 404 rather than leaking its existence.
-- name: GetRequestForBuyer :one
SELECT r.*
FROM quota_request r
JOIN business_profile p ON p.id = r.buyer_id
WHERE r.id = $1 AND p.account_id = $2;

-- ListOffersByRequest returns every offer across all candidates of a request so
-- the detail view attaches each candidate's chain in one query (FR-032).
-- name: ListOffersByRequest :many
SELECT o.id, o.candidate_id, o.sequence, o.proposed_by, o.total_price,
       o.readiness_lead_days, o.note, o.created_at
FROM offer o
JOIN request_candidate c ON c.id = o.candidate_id
WHERE c.request_id = $1
ORDER BY o.candidate_id, o.sequence ASC;

-- ListOffersByCandidates returns every offer for the given candidates so the
-- incoming list attaches each candidate's chain in one query (FR-032, FR-033).
-- The subcontractor needs the buyer's latest counter round to reply, which lives
-- only in the offer chain, not on the candidate row.
-- name: ListOffersByCandidates :many
SELECT o.id, o.candidate_id, o.sequence, o.proposed_by, o.total_price,
       o.readiness_lead_days, o.note, o.created_at
FROM offer o
WHERE o.candidate_id = ANY($1::uuid[])
ORDER BY o.candidate_id, o.sequence ASC;

-- GetIncomingCandidate loads one candidate for its subcontractor owner. The
-- account guard makes a copied detail link look like a missing candidate to a
-- different subcontractor, rather than exposing whether the id exists (FR-030).
-- It carries the same request and capacity fields as ListIncomingCandidates so a
-- browser refresh does not need a cached list page.
-- name: GetIncomingCandidate :one
SELECT c.id AS candidate_id, c.listing_id, c.subcontractor_id,
       c.status, c.rejection_reason, p.business_name,
       r.quantity, r.material, r.deadline, r.note,
       l.weekly_capacity, l.readiness_lead_days, l.horizon_until
FROM request_candidate c
JOIN capacity_listing l ON l.id = c.listing_id
JOIN business_profile owner ON owner.id = l.profile_id
JOIN quota_request r ON r.id = c.request_id
JOIN business_profile p ON p.id = r.buyer_id
WHERE c.id = $1 AND owner.account_id = $2;

-- ListIncomingCandidates returns one keyset page of candidates whose listing the
-- subcontractor account owns, newest request first (FR-030). An optional status
-- filter narrows to one candidate_status. The cursor tuple is (created_at, id)
-- of the request, matching the buyer-side list. It also carries the request's
-- quantity, material, note, and deadline plus the listing's capacity shape
-- (weekly_capacity, readiness_lead_days, horizon_until) so the read side can mark
-- whether the subcontractor can fulfil each request within its readiness..deadline
-- range (FR-035, FR-090) without a second query per row.
-- name: ListIncomingCandidates :many
SELECT c.id AS candidate_id, c.request_id, c.listing_id, c.subcontractor_id,
       c.status, c.rejection_reason, p.business_name,
       r.created_at, r.quantity, r.material, r.note, r.deadline,
       l.weekly_capacity, l.readiness_lead_days, l.horizon_until
FROM request_candidate c
JOIN capacity_listing l ON l.id = c.listing_id
JOIN business_profile owner ON owner.id = l.profile_id
JOIN quota_request r ON r.id = c.request_id
JOIN business_profile p ON p.id = r.buyer_id
WHERE owner.account_id = $1
  AND (sqlc.narg('status')::candidate_status IS NULL OR c.status = sqlc.narg('status')::candidate_status)
  AND (r.created_at, r.id) < ($2::timestamptz, $3::uuid)
ORDER BY r.created_at DESC, r.id DESC
LIMIT $4;
