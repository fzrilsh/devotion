CREATE TYPE work_order_status AS ENUM (
    'accepted', 'production', 'completed', 'shipped', 'confirmed', 'cancelled', 'in_mediation'
);

CREATE TABLE work_order (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id        uuid NOT NULL REFERENCES request_candidate(id) ON DELETE RESTRICT,
    offer_id            uuid NOT NULL REFERENCES offer(id) ON DELETE RESTRICT,
    buyer_id            uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    subcontractor_id    uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    quantity            integer NOT NULL,
    total_price         bigint NOT NULL,
    deadline            date NOT NULL,
    readiness_week_start date NOT NULL,
    status              work_order_status NOT NULL DEFAULT 'accepted',
    shipped_at          timestamptz,
    confirmed_at        timestamptz,
    auto_confirmed      boolean NOT NULL DEFAULT false,
    cancelled_by_id     uuid REFERENCES business_profile(id),
    cancellation_reason text,
    cancelled_at        timestamptz,
    created_at          timestamptz NOT NULL,

    CONSTRAINT one_order_per_candidate UNIQUE (candidate_id),
    CONSTRAINT two_distinct_parties CHECK (buyer_id <> subcontractor_id),
    CONSTRAINT order_quantity_positive CHECK (quantity > 0),
    CONSTRAINT order_price_positive CHECK (total_price > 0),
    CONSTRAINT readiness_is_monday CHECK (EXTRACT(ISODOW FROM readiness_week_start) = 1),
    CONSTRAINT readiness_not_past_deadline CHECK (readiness_week_start <= deadline),
    CONSTRAINT cancellation_complete CHECK (
        (status <> 'cancelled')
        OR (cancelled_by_id IS NOT NULL AND cancellation_reason IS NOT NULL
            AND cancelled_at IS NOT NULL)
    ),
    CONSTRAINT shipped_before_confirmed CHECK (
        confirmed_at IS NULL OR shipped_at IS NULL OR confirmed_at >= shipped_at
    ),
    CONSTRAINT auto_confirm_needs_confirmation CHECK (
        NOT auto_confirmed OR confirmed_at IS NOT NULL
    )
);

CREATE INDEX idx_order_buyer ON work_order (buyer_id, status);
CREATE INDEX idx_order_subcon ON work_order (subcontractor_id, status);
CREATE INDEX idx_order_deadline_active ON work_order (deadline)
    WHERE status IN ('accepted', 'production', 'completed', 'shipped');
CREATE INDEX idx_order_auto_confirm ON work_order (shipped_at) WHERE status = 'shipped';

CREATE TABLE work_order_status_history (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id uuid NOT NULL REFERENCES work_order(id) ON DELETE CASCADE,
    old_status   work_order_status,
    new_status   work_order_status NOT NULL,
    changed_by   uuid REFERENCES user_account(id),
    by_system    boolean NOT NULL DEFAULT false,
    note         text,
    created_at   timestamptz NOT NULL,

    CONSTRAINT actor_clear CHECK (by_system OR changed_by IS NOT NULL)
);

CREATE INDEX idx_status_history_order ON work_order_status_history (work_order_id, created_at);

CREATE TYPE payment_direction AS ENUM ('sent', 'received');

CREATE TABLE payment_record (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id uuid NOT NULL REFERENCES work_order(id) ON DELETE CASCADE,
    profile_id   uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    direction    payment_direction NOT NULL,
    date         date NOT NULL,
    note         text,
    created_at   timestamptz NOT NULL,

    CONSTRAINT one_statement_per_party_per_direction UNIQUE (work_order_id, profile_id, direction)
);

CREATE INDEX idx_payment_order ON payment_record (work_order_id);
