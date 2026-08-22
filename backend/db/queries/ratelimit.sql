-- TouchRateLimit increments the counter for one (target, key) in the current
-- window bucket, creating the row on first use, and returns the new count. The
-- ON CONFLICT DO UPDATE is atomic and takes a row lock, so two concurrent
-- callers cannot both read the same count and both slip through: the second
-- serializes behind the first and sees the incremented value.
-- name: TouchRateLimit :one
INSERT INTO rate_limit (id, target, key, window_start, count)
VALUES (gen_random_uuid(), $1, $2, $3, 1)
ON CONFLICT (target, key, window_start)
DO UPDATE SET count = rate_limit.count + 1
RETURNING count;

-- LockRateLimitKey takes a transaction-scoped advisory lock so the distinct
-- counting path (otp_address) serializes per source address. Without it, two
-- new numbers from the same address could both pass the distinct-count check.
-- name: LockRateLimitKey :exec
SELECT pg_advisory_xact_lock($1);

-- MemberRecorded reports whether member was already recorded under key in the
-- current window. A re-send to a number already counted is not a new distinct
-- number, so it does not consume more of the address budget.
-- name: MemberRecorded :one
SELECT EXISTS (
    SELECT 1 FROM rate_limit
    WHERE target = $1 AND key = $2 AND window_start = $3
);

-- CountDistinctMembers counts the distinct members recorded under an address in
-- the current window. Each (address, member) pair is one row, so the row count
-- is the distinct-member count. The pattern is address + separator + '%'.
-- name: CountDistinctMembers :one
SELECT count(*) FROM rate_limit
WHERE target = $1 AND key LIKE $2 AND window_start = $3;

-- RecordMember records that member was used under key in the current window.
-- DO NOTHING keeps it idempotent within the window.
-- name: RecordMember :exec
INSERT INTO rate_limit (id, target, key, window_start, count)
VALUES (gen_random_uuid(), $1, $2, $3, 1)
ON CONFLICT (target, key, window_start) DO NOTHING;
