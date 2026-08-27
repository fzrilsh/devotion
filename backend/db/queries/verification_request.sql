-- CreateVerificationRequest records one pending verification submission for a
-- profile. status defaults to 'pending' in the schema, so it is not set here;
-- decided_by, decided_at and admin_note stay NULL until an admin decides. The
-- idx_one_pending_verification partial unique index rejects a second pending row
-- for the same profile with a 23505, which the handler turns into a 409
-- (FR-011: re-submission is allowed only after a rejection).
-- name: CreateVerificationRequest :one
INSERT INTO verification_request (
    id, profile_id, identity_number, identity_file_id, location_file_id,
    applicant_source_address, created_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- ListVerificationRequestsByProfile returns every submission a profile has made,
-- newest first, joined to business_profile for the business_name the contract's
-- VerificationRequest carries (the verification_request table has no such
-- column). The caller sees only their own submissions (FR-006).
-- name: ListVerificationRequestsByProfile :many
SELECT
    vr.id, vr.profile_id, vr.identity_number, vr.identity_file_id,
    vr.location_file_id, vr.status, vr.admin_note, vr.decided_at, vr.created_at,
    bp.business_name
FROM verification_request vr
JOIN business_profile bp ON bp.id = vr.profile_id
WHERE vr.profile_id = $1
ORDER BY vr.created_at DESC;

-- ListVerificationQueue returns one keyset page of verification requests for the
-- admin queue, newest first, joined to business_profile for the business name
-- the contract's VerificationRequest carries (FR-007). An absent status_filter
-- lists every status; a present one narrows to it, so the default view is the
-- pending backlog. A null before_created is the first page; the (created_at, id)
-- tuple comparison is the stable keyset the opaque cursor rides (FR-008 badge is
-- granted from the decision this queue feeds, not here).
-- name: ListVerificationQueue :many
SELECT
    vr.id, vr.profile_id, vr.identity_number, vr.identity_file_id,
    vr.location_file_id, vr.status, vr.admin_note, vr.decided_at, vr.created_at,
    bp.business_name
FROM verification_request vr
JOIN business_profile bp ON bp.id = vr.profile_id
WHERE (
        sqlc.narg(status_filter)::verification_status IS NULL
        OR vr.status = sqlc.narg(status_filter)::verification_status
    )
  AND (
        sqlc.narg(before_created)::timestamptz IS NULL
        OR (vr.created_at, vr.id) < (sqlc.narg(before_created)::timestamptz, sqlc.narg(before_id)::uuid)
    )
ORDER BY vr.created_at DESC, vr.id DESC
LIMIT sqlc.arg(page_limit)::int;

-- LockVerificationRequest takes a row lock on the target request so the decision
-- runs under a lock: two admins deciding the same pending request serialize, and
-- the second sees the status the first already set. business_name rides along via
-- a scalar subquery so the decision response carries it without a second round
-- trip (FR-007).
-- name: LockVerificationRequest :one
SELECT vr.*, bp.business_name
FROM verification_request vr
JOIN business_profile bp ON bp.id = vr.profile_id
WHERE vr.id = sqlc.arg(id)::uuid
FOR UPDATE OF vr;

-- DecideVerificationRequest records an admin's approval or rejection: it stamps
-- status, admin_note, decided_by and decided_at in one statement, only while the
-- request is still pending (FR-007). The verification_decision_complete and
-- rejection_needs_reason CHECKs are the storage-level safety net; the handler
-- rejects a reasonless rejection first so the applicant reads a field error, not
-- a 500. business_name rides along for the response.
-- name: DecideVerificationRequest :one
UPDATE verification_request
SET status = sqlc.arg(status)::verification_status,
    admin_note = sqlc.narg(admin_note),
    decided_by = sqlc.arg(decided_by)::uuid,
    decided_at = sqlc.arg(decided_at)::timestamptz
WHERE id = sqlc.arg(id)::uuid AND status = 'pending'
RETURNING *,
    (SELECT business_name FROM business_profile WHERE id = profile_id) AS business_name;

-- MarkProfileVerified flips the verified badge on approval (FR-008). It is called
-- inside the same transaction as DecideVerificationRequest so an approved
-- decision and the badge it grants land together or not at all. search.sql
-- already selects verified, so the badge propagates to search results with no
-- further change; a rejection never touches this, leaving the listing untouched
-- (FR-010, FR-011).
-- name: MarkProfileVerified :exec
UPDATE business_profile
SET verified = true, updated_at = sqlc.arg(updated_at)::timestamptz
WHERE id = sqlc.arg(id)::uuid;
