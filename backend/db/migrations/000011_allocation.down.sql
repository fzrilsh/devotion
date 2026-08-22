DROP TRIGGER IF EXISTS trg_reject_allocation_before_readiness ON capacity_allocation;
DROP FUNCTION IF EXISTS reject_allocation_before_readiness();
DROP INDEX IF EXISTS idx_allocation_period;
DROP INDEX IF EXISTS idx_allocation_order;
DROP TABLE IF EXISTS capacity_allocation;
