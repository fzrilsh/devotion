-- auto_confirm_base_at is the instant the 7-day auto-confirm clock counts from
-- (FR-068), normally NULL. shipped_at records when the order was shipped and is a
-- historical fact that never moves; when a dispute is mediated and the outcome is
-- "continued" on an already-shipped order, the auto-confirm clock must restart
-- from the moment mediation closed (data-model.md), not from the original
-- shipment. Rather than overwrite shipped_at (which would erase the shipment
-- fact), the resolve path stamps auto_confirm_base_at with the mediation-close
-- instant, and the auto-confirm arithmetic reads COALESCE(auto_confirm_base_at,
-- shipped_at) as its effective base. NULL everywhere else means the base is
-- shipped_at, so orders that were never mediated behave exactly as before.
ALTER TABLE work_order ADD COLUMN auto_confirm_base_at timestamptz;

-- The base can only move forward from shipment: mediation always happens after an
-- order is shipped, so a base earlier than shipped_at would be a time-ordering
-- bug. This mirrors shipped_before_confirmed, catching a bad restart instant at
-- the storage layer instead of letting it silently shorten the window.
ALTER TABLE work_order ADD CONSTRAINT auto_confirm_base_after_shipped CHECK (
    auto_confirm_base_at IS NULL OR shipped_at IS NULL
    OR auto_confirm_base_at >= shipped_at
);

-- The auto-confirm scans key off the effective base, not shipped_at, so the
-- partial index that backs them (idx_order_auto_confirm) must cover the same
-- expression. A shifted-base order still sits under status='shipped', so the
-- partial predicate is unchanged; only the indexed expression becomes the COALESCE
-- so the ticker's ORDER BY and range test ride the index rather than sorting.
DROP INDEX idx_order_auto_confirm;
CREATE INDEX idx_order_auto_confirm
    ON work_order (COALESCE(auto_confirm_base_at, shipped_at))
    WHERE status = 'shipped';
