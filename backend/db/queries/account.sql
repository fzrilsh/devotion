-- CreateAccount inserts a new account. business role columns default false at
-- the table level; the caller passes only the roles chosen at registration.
-- password_hash is a bcrypt digest, never the plaintext password.
-- name: CreateAccount :one
INSERT INTO user_account (
    id, email, phone, password_hash,
    role_subcontractor, role_buyer,
    created_at, updated_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $6
)
RETURNING *;

-- UpsertAdmin creates the admin account or, when the email already exists,
-- resets its password. Idempotent so admin:create can run twice without a
-- duplicate. role_admin is set true and the two business roles false, which the
-- admin_has_no_business_role and has_at_least_one_role constraints both accept.
-- name: UpsertAdmin :one
INSERT INTO user_account (
    id, email, phone, password_hash,
    role_subcontractor, role_buyer, role_admin,
    email_verified, phone_verified,
    created_at, updated_at
) VALUES (
    gen_random_uuid(), $1, $2, $3,
    false, false, true,
    true, true,
    $4, $4
)
ON CONFLICT (email) DO UPDATE
SET password_hash = EXCLUDED.password_hash,
    updated_at    = EXCLUDED.updated_at
RETURNING *;

-- GetAccountByEmail loads one account by its case-insensitive email. Used by
-- login and recovery; both must run the rate limit before this lookup.
-- name: GetAccountByEmail :one
SELECT * FROM user_account WHERE email = $1;

-- GetAccountByID loads one account by id, backing GET /me and session refresh.
-- name: GetAccountByID :one
SELECT * FROM user_account WHERE id = $1;

-- GetAccountByPhone loads one account by phone, backing WhatsApp verification
-- and the emergency user:verify --phone subcommand.
-- name: GetAccountByPhone :one
SELECT * FROM user_account WHERE phone = $1;

-- EmailExists reports whether an email is already registered, so registration
-- can answer 409 without leaking through a full row read.
-- name: EmailExists :one
SELECT EXISTS (SELECT 1 FROM user_account WHERE email = $1);

-- PhoneExists reports whether a phone is already registered.
-- name: PhoneExists :one
SELECT EXISTS (SELECT 1 FROM user_account WHERE phone = $1);

-- SetEmailVerified marks the email verified after a valid code is consumed.
-- name: SetEmailVerified :exec
UPDATE user_account
SET email_verified = true, updated_at = $2
WHERE id = $1;

-- SetPhoneVerified marks the phone verified after a valid code is consumed.
-- name: SetPhoneVerified :exec
UPDATE user_account
SET phone_verified = true, updated_at = $2
WHERE id = $1;

-- UpdatePassword replaces the bcrypt hash during recovery confirmation.
-- name: UpdatePassword :exec
UPDATE user_account
SET password_hash = $2, updated_at = $3
WHERE id = $1;

-- GetProfileIDByAccount returns the business profile id for an account, or no
-- rows when none exists yet. MyAccount carries profile_id as nullable, so the
-- caller treats pgx.ErrNoRows as a null profile_id rather than an error.
-- name: GetProfileIDByAccount :one
SELECT id FROM business_profile WHERE account_id = $1;

-- CountActiveOrdersAsBuyer counts a profile's orders that are not terminal, on
-- the buyer side. Revoking the buyer role while any of these exist is refused,
-- because it would strip a party from an order still in flight.
-- name: CountActiveOrdersAsBuyer :one
SELECT count(*) FROM work_order
WHERE buyer_id = $1
  AND status IN ('accepted', 'production', 'completed', 'shipped', 'in_mediation');

-- CountActiveOrdersAsSubcontractor counts a profile's non-terminal orders on the
-- subcontractor side, guarding subcontractor-role revocation the same way.
-- name: CountActiveOrdersAsSubcontractor :one
SELECT count(*) FROM work_order
WHERE subcontractor_id = $1
  AND status IN ('accepted', 'production', 'completed', 'shipped', 'in_mediation');

-- UpdateBusinessRoles sets the two business role flags. admin is never touched
-- here; the admin_has_no_business_role constraint rejects granting a business
-- role to an admin account, so the update fails loudly instead of silently
-- corrupting the role model.
-- name: UpdateBusinessRoles :one
UPDATE user_account
SET role_subcontractor = $2, role_buyer = $3, updated_at = $4
WHERE id = $1
RETURNING *;
