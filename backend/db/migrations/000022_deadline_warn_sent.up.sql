-- deadline_warn_sent_at records the moment the FR-051 "delivery deadline
-- approaching" notice was sent to both parties of an active order. It is the
-- dedup marker for the in-process ticker (research.md R-07 layer 2): the scan for
-- orders needing the warning filters on deadline_warn_sent_at IS NULL, so the
-- 5-minute ticker warns each order once across the 7-day lead rather than on
-- every tick. It stays NULL for orders whose deadline is still further out than
-- the lead, and is never cleared once stamped.
ALTER TABLE work_order ADD COLUMN deadline_warn_sent_at timestamptz;

-- idx_order_deadline_warn serves the FR-051 approaching scan: only active,
-- pre-shipment orders not yet warned, ordered by deadline. It mirrors
-- idx_order_confirm_warn's partial shape so the ticker touches a small hot set.
CREATE INDEX idx_order_deadline_warn ON work_order (deadline)
    WHERE status IN ('accepted', 'production', 'completed') AND deadline_warn_sent_at IS NULL;
