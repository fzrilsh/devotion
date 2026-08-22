CREATE TYPE verification_purpose AS ENUM ('email', 'phone', 'recovery');

CREATE TABLE verification_code (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  uuid NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    purpose     verification_purpose NOT NULL,
    code_hash   bytea NOT NULL,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL
);

CREATE INDEX idx_verification_code_lookup ON verification_code (account_id, purpose);
