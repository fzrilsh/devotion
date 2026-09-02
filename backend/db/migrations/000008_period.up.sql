CREATE TABLE availability_period (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id     uuid NOT NULL REFERENCES capacity_listing(id) ON DELETE CASCADE,
    week_start     date NOT NULL,
    total_capacity integer NOT NULL,
    used_capacity  integer NOT NULL DEFAULT 0,
    marked_full    boolean NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL,

    CONSTRAINT one_period_per_week UNIQUE (listing_id, week_start),
    CONSTRAINT week_start_is_monday CHECK (EXTRACT(ISODOW FROM week_start) = 1),
    CONSTRAINT total_capacity_not_negative CHECK (total_capacity >= 0),
    CONSTRAINT used_capacity_within_total CHECK (
        used_capacity >= 0 AND used_capacity <= total_capacity
    )
);

CREATE INDEX idx_period_listing_week ON availability_period (listing_id, week_start);
CREATE INDEX idx_period_available ON availability_period (listing_id, week_start)
    WHERE NOT marked_full AND used_capacity < total_capacity;
