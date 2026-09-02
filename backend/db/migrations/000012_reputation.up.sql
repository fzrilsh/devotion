CREATE TABLE review (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id    uuid NOT NULL REFERENCES work_order(id) ON DELETE RESTRICT,
    reviewer_id      uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    reviewee_id      uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    rating           smallint NOT NULL,
    text             text,
    hidden           boolean NOT NULL DEFAULT false,
    hidden_by        uuid REFERENCES user_account(id),
    hidden_at        timestamptz,
    hidden_reason    text,
    created_at       timestamptz NOT NULL,

    CONSTRAINT one_review_per_order_per_reviewer UNIQUE (work_order_id, reviewer_id),
    CONSTRAINT rating_one_to_five CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT no_self_review CHECK (reviewer_id <> reviewee_id),
    CONSTRAINT hiding_complete CHECK (
        NOT hidden
        OR (hidden_by IS NOT NULL AND hidden_at IS NOT NULL
            AND hidden_reason IS NOT NULL)
    )
);

CREATE INDEX idx_review_reviewee ON review (reviewee_id) WHERE NOT hidden;
CREATE INDEX idx_review_order ON review (work_order_id);

CREATE TYPE dispute_status AS ENUM ('reported', 'in_mediation', 'resolved');

CREATE TABLE dispute (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id      uuid NOT NULL REFERENCES work_order(id) ON DELETE RESTRICT,
    reporter_id        uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    report_body        text NOT NULL,
    status             dispute_status NOT NULL DEFAULT 'reported',
    admin_note         text,
    allocation_reversed boolean,
    liable_party_id    uuid REFERENCES business_profile(id),
    handled_by         uuid REFERENCES user_account(id),
    resolved_at        timestamptz,
    created_at         timestamptz NOT NULL,

    CONSTRAINT resolution_complete CHECK (
        status <> 'resolved'
        OR (handled_by IS NOT NULL AND resolved_at IS NOT NULL
            AND allocation_reversed IS NOT NULL AND admin_note IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_one_open_dispute
    ON dispute (work_order_id) WHERE status <> 'resolved';
CREATE INDEX idx_dispute_queue ON dispute (created_at) WHERE status <> 'resolved';
