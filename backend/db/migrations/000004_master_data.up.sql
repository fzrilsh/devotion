CREATE TYPE item_type AS ENUM ('product', 'machine');

CREATE TABLE catalog_item (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    type       item_type NOT NULL,
    name       text NOT NULL,
    active     boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL,

    CONSTRAINT item_name_unique_per_type UNIQUE (type, name)
);

CREATE INDEX idx_item_active ON catalog_item (type, active) WHERE active;
