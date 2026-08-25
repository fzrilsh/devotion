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
