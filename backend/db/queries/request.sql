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
