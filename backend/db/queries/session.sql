-- CreateSession stores a new session row keyed by the SHA-256 hash of the
-- opaque token. The raw token lives only in the cookie; the database never
-- sees it, so a database read cannot reconstruct a usable session token.
-- name: CreateSession :one
INSERT INTO session (
    id, account_id, token_hash, source_address,
    expires_at, created_at, accessed_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $5
)
RETURNING *;

-- GetSessionByTokenHash loads a live session by token hash. The expiry check is
-- in SQL so an expired row is treated as absent without a second round trip.
-- name: GetSessionByTokenHash :one
SELECT * FROM session
WHERE token_hash = $1 AND expires_at > $2;

-- RenewSession slides the expiry forward and records access time, implementing
-- rolling 7-day renewal on each authenticated request.
-- name: RenewSession :exec
UPDATE session
SET expires_at = $2, accessed_at = $3
WHERE id = $1;

-- DeleteSession removes one session by token hash, backing logout.
-- name: DeleteSession :exec
DELETE FROM session WHERE token_hash = $1;

-- DeleteOtherSessions removes every session for an account except the one whose
-- hash is kept. Recovery confirmation uses this to end all other sessions after
-- a password reset, without logging the current caller out mid-request.
-- name: DeleteOtherSessions :exec
DELETE FROM session
WHERE account_id = $1 AND token_hash <> $2;

-- DeleteAllSessions removes every session for an account, used when no session
-- is retained (recovery confirmed without an active caller session).
-- name: DeleteAllSessions :exec
DELETE FROM session WHERE account_id = $1;
