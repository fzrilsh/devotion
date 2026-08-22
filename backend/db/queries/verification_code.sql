-- CreateVerificationCode stores one six-digit code as its SHA-256 hash for a
-- given account and purpose. The plaintext code is delivered out of band (email
-- or WhatsApp) and never persisted, so a database read cannot reveal it.
-- name: CreateVerificationCode :one
INSERT INTO verification_code (
    id, account_id, purpose, code_hash, expires_at, created_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5
)
RETURNING *;

-- GetLatestVerificationCode returns the most recent unconsumed code for an
-- account and purpose. Verification compares the submitted code's hash against
-- this row and checks expiry in Go against the injected Clock.
-- name: GetLatestVerificationCode :one
SELECT * FROM verification_code
WHERE account_id = $1 AND purpose = $2 AND consumed_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- ConsumeVerificationCode marks a code used so it cannot be replayed. The
-- consumed_at IS NULL guard makes double-consume a no-op, and the returned row
-- count lets the caller detect a code already spent.
-- name: ConsumeVerificationCode :execrows
UPDATE verification_code
SET consumed_at = $2
WHERE id = $1 AND consumed_at IS NULL;

-- InvalidateVerificationCodes consumes all outstanding codes for an account and
-- purpose, so issuing a fresh code retires the previous ones in the same
-- transaction and only the newest can be redeemed.
-- name: InvalidateVerificationCodes :exec
UPDATE verification_code
SET consumed_at = $3
WHERE account_id = $1 AND purpose = $2 AND consumed_at IS NULL;
