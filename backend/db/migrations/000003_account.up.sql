CREATE TABLE user_account (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email                citext NOT NULL,
    phone                text   NOT NULL,
    password_hash        text   NOT NULL,
    email_verified       boolean NOT NULL DEFAULT false,
    phone_verified       boolean NOT NULL DEFAULT false,
    role_subcontractor   boolean NOT NULL DEFAULT false,
    role_buyer           boolean NOT NULL DEFAULT false,
    role_admin           boolean NOT NULL DEFAULT false,
    notif_nontx_email    boolean NOT NULL DEFAULT true,
    notif_nontx_whatsapp boolean NOT NULL DEFAULT true,
    created_at           timestamptz NOT NULL,
    updated_at           timestamptz NOT NULL,

    CONSTRAINT email_unique UNIQUE (email),
    CONSTRAINT phone_unique UNIQUE (phone),
    CONSTRAINT phone_format CHECK (phone ~ '^62[0-9]{8,13}$'),
    CONSTRAINT has_at_least_one_role CHECK (
        role_subcontractor OR role_buyer OR role_admin
    ),
    CONSTRAINT admin_has_no_business_role CHECK (
        NOT role_admin OR (NOT role_subcontractor AND NOT role_buyer)
    )
);

CREATE TABLE session (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id     uuid NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    token_hash     bytea NOT NULL,
    source_address inet,
    expires_at     timestamptz NOT NULL,
    created_at     timestamptz NOT NULL,
    accessed_at    timestamptz NOT NULL,

    CONSTRAINT token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX idx_session_account ON session (account_id);
CREATE INDEX idx_session_expires ON session (expires_at);
