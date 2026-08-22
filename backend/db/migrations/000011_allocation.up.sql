CREATE TABLE capacity_allocation (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id uuid NOT NULL REFERENCES work_order(id) ON DELETE RESTRICT,
    period_id    uuid NOT NULL REFERENCES availability_period(id) ON DELETE RESTRICT,
    quantity     integer NOT NULL,
    created_at   timestamptz NOT NULL,
    reversed_at  timestamptz,

    CONSTRAINT one_allocation_per_order_period UNIQUE (work_order_id, period_id),
    CONSTRAINT allocation_quantity_positive CHECK (quantity > 0)
);

CREATE INDEX idx_allocation_order ON capacity_allocation (work_order_id);
CREATE INDEX idx_allocation_period ON capacity_allocation (period_id) WHERE reversed_at IS NULL;

CREATE FUNCTION reject_allocation_before_readiness() RETURNS trigger AS $$
DECLARE
    v_period_week    date;
    v_readiness_week date;
BEGIN
    SELECT p.week_start INTO v_period_week
      FROM availability_period p WHERE p.id = NEW.period_id;

    SELECT o.readiness_week_start INTO v_readiness_week
      FROM work_order o WHERE o.id = NEW.work_order_id;

    IF v_period_week < v_readiness_week THEN
        RAISE EXCEPTION
            'FR-087: allocation on week % precedes readiness start week %',
            v_period_week, v_readiness_week;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_reject_allocation_before_readiness
    BEFORE INSERT OR UPDATE ON capacity_allocation
    FOR EACH ROW EXECUTE FUNCTION reject_allocation_before_readiness();
