CREATE TYPE file_type AS ENUM ('identity_document', 'location_photo');

CREATE TABLE uploaded_file (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_profile_id  uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    type              file_type NOT NULL,
    original_name     text NOT NULL,
    mime_type         text NOT NULL,
    size_bytes        integer NOT NULL,
    storage_path      text NOT NULL,
    created_at        timestamptz NOT NULL,

    CONSTRAINT max_size CHECK (size_bytes > 0 AND size_bytes <= 5 * 1024 * 1024),
    CONSTRAINT allowed_type CHECK (mime_type IN ('image/jpeg', 'image/png', 'application/pdf')),
    CONSTRAINT storage_path_unique UNIQUE (storage_path)
);

CREATE INDEX idx_file_owner ON uploaded_file (owner_profile_id);

CREATE TYPE verification_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE verification_request (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id           uuid NOT NULL REFERENCES business_profile(id) ON DELETE CASCADE,
    identity_number      text NOT NULL,
    identity_file_id     uuid NOT NULL REFERENCES uploaded_file(id) ON DELETE RESTRICT,
    location_file_id     uuid NOT NULL REFERENCES uploaded_file(id) ON DELETE RESTRICT,
    status               verification_status NOT NULL DEFAULT 'pending',
    admin_note           text,
    decided_by           uuid REFERENCES user_account(id),
    decided_at           timestamptz,
    applicant_source_address inet,
    created_at           timestamptz NOT NULL,

    CONSTRAINT verification_decision_complete CHECK (
        (status = 'pending' AND decided_at IS NULL)
        OR (status <> 'pending' AND decided_at IS NOT NULL AND decided_by IS NOT NULL)
    ),
    CONSTRAINT rejection_needs_reason CHECK (status <> 'rejected' OR admin_note IS NOT NULL)
);

CREATE UNIQUE INDEX idx_one_pending_verification
    ON verification_request (profile_id) WHERE status = 'pending';
CREATE INDEX idx_verification_queue
    ON verification_request (created_at) WHERE status = 'pending';
