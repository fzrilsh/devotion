DROP INDEX idx_order_auto_confirm;
CREATE INDEX idx_order_auto_confirm ON work_order (shipped_at) WHERE status = 'shipped';

ALTER TABLE work_order DROP CONSTRAINT auto_confirm_base_after_shipped;
ALTER TABLE work_order DROP COLUMN auto_confirm_base_at;
