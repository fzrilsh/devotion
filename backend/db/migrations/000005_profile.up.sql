CREATE TABLE business_profile (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    uuid NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
    business_name text NOT NULL,
    city_code     text NOT NULL REFERENCES city(code) ON DELETE RESTRICT,
    latitude      numeric(9,6),
    longitude     numeric(9,6),
    description   text,
    verified      boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL,
    updated_at    timestamptz NOT NULL,

    CONSTRAINT one_profile_per_account UNIQUE (account_id),
    CONSTRAINT business_name_not_empty CHECK (length(trim(business_name)) >= 3),
    CONSTRAINT coordinates_complete_or_empty CHECK (
        (latitude IS NULL AND longitude IS NULL) OR (latitude IS NOT NULL AND longitude IS NOT NULL)
    ),
    CONSTRAINT coordinates_within_indonesia CHECK (
        latitude IS NULL OR (latitude BETWEEN -11.5 AND 6.5 AND longitude BETWEEN 94.5 AND 141.5)
    )
);

CREATE INDEX idx_profile_city ON business_profile (city_code);
CREATE INDEX idx_profile_name ON business_profile (business_name);

CREATE TYPE proposal_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE item_proposal (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id    uuid NOT NULL REFERENCES business_profile(id) ON DELETE CASCADE,
    type          item_type NOT NULL,
    proposed_name text NOT NULL,
    status        proposal_status NOT NULL DEFAULT 'pending',
    admin_note    text,
    decided_by    uuid REFERENCES user_account(id),
    decided_at    timestamptz,
    item_id       uuid REFERENCES catalog_item(id),
    created_at    timestamptz NOT NULL,

    CONSTRAINT decision_complete CHECK (
        (status = 'pending' AND decided_at IS NULL AND decided_by IS NULL)
        OR (status <> 'pending' AND decided_at IS NOT NULL AND decided_by IS NOT NULL)
    ),
    CONSTRAINT approved_yields_item CHECK (status <> 'approved' OR item_id IS NOT NULL)
);

CREATE INDEX idx_proposal_pending ON item_proposal (created_at) WHERE status = 'pending';
