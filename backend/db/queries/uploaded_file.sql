-- CreateUploadedFile records one stored file. The storage_path is a
-- system-generated UUID name, never the client's filename; original_name is
-- kept as display metadata only. size_bytes and mime_type are the values the
-- application verified (magic bytes, re-encoded length), not what the client
-- claimed, so the DB CHECK constraints see the trusted values.
-- name: CreateUploadedFile :one
INSERT INTO uploaded_file (
    id, owner_profile_id, type, original_name, mime_type, size_bytes, storage_path, created_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- GetUploadedFile returns one file row by id. The handler resolves the caller's
-- profile and admin flag and compares against owner_profile_id before streaming
-- the bytes, so this query carries no access check of its own (FR-009 is
-- enforced in Go, not SQL).
-- name: GetUploadedFile :one
SELECT * FROM uploaded_file WHERE id = $1;

-- SumUploadedBytesByOwner totals the bytes a profile already holds, so a new
-- upload can be rejected before it pushes the owner past the 500MB quota.
-- COALESCE keeps the result 0 rather than NULL for an owner with no files yet.
-- name: SumUploadedBytesByOwner :one
SELECT COALESCE(SUM(size_bytes), 0)::bigint FROM uploaded_file WHERE owner_profile_id = $1;
