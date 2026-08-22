CREATE TABLE capacity_listing (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id          uuid NOT NULL REFERENCES business_profile(id) ON DELETE CASCADE,
    weekly_capacity     integer NOT NULL,
    readiness_lead_days integer NOT NULL,
    published           boolean NOT NULL DEFAULT true,
    calendar_updated_at timestamptz NOT NULL,
    horizon_until       date NOT NULL,
    created_at          timestamptz NOT NULL,
    updated_at          timestamptz NOT NULL,

    CONSTRAINT one_listing_per_profile UNIQUE (profile_id),
    CONSTRAINT capacity_positive CHECK (weekly_capacity > 0),
    CONSTRAINT lead_not_negative CHECK (readiness_lead_days >= 0 AND readiness_lead_days <= 365),
    CONSTRAINT horizon_is_monday CHECK (EXTRACT(ISODOW FROM horizon_until) = 1)
);

CREATE INDEX idx_listing_published ON capacity_listing (id) WHERE published;
CREATE INDEX idx_listing_calendar_stale ON capacity_listing (calendar_updated_at) WHERE published;
CREATE INDEX idx_listing_horizon ON capacity_listing (horizon_until) WHERE published;

CREATE TABLE listing_product (
    listing_id uuid NOT NULL REFERENCES capacity_listing(id) ON DELETE CASCADE,
    item_id    uuid NOT NULL REFERENCES catalog_item(id) ON DELETE RESTRICT,
    PRIMARY KEY (listing_id, item_id)
);

CREATE TABLE listing_machine (
    listing_id    uuid NOT NULL REFERENCES capacity_listing(id) ON DELETE CASCADE,
    item_id       uuid NOT NULL REFERENCES catalog_item(id) ON DELETE RESTRICT,
    machine_count integer NOT NULL,
    PRIMARY KEY (listing_id, item_id),
    CONSTRAINT machine_count_positive CHECK (machine_count > 0)
);

CREATE INDEX idx_listing_product_item ON listing_product (item_id);
CREATE INDEX idx_listing_machine_item ON listing_machine (item_id);

CREATE FUNCTION reject_wrong_item_type() RETURNS trigger AS $$
DECLARE
    v_type          text;
    v_expected_type text := TG_ARGV[0];
BEGIN
    SELECT i.type INTO v_type
      FROM catalog_item i WHERE i.id = NEW.item_id;

    IF v_type IS DISTINCT FROM v_expected_type THEN
        RAISE EXCEPTION
            'item % has type %, this table only accepts type %',
            NEW.item_id, v_type, v_expected_type;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_reject_wrong_product_item
    BEFORE INSERT OR UPDATE ON listing_product
    FOR EACH ROW EXECUTE FUNCTION reject_wrong_item_type('product');

CREATE TRIGGER trg_reject_wrong_machine_item
    BEFORE INSERT OR UPDATE ON listing_machine
    FOR EACH ROW EXECUTE FUNCTION reject_wrong_item_type('machine');
