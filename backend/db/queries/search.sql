-- SearchCandidates ranks published listings against the four hard criteria
-- (FR-023..FR-025) and returns one keyset page. The query shape follows
-- data-model.md section 10: score lives in the `ranked` CTE so the keyset WHERE
-- can filter on it, capacity is summed across the readiness..deadline range
-- (FR-080), and periods past horizon_until are counted optimistically as full
-- (FR-088). The application rounds the deadline to Monday before calling this,
-- so Go stays the single source of truth for that rounding, and horizon
-- extension happens outside this read query for passing candidates only. A NULL
-- product/machine/lead filter counts as satisfied (FR-023, decision C-4) so the
-- score stays 0..4 with no weighting or normalization (FR-024). The keyset tuple
-- ends in listing_id to make the order total and repeatable across pages.
-- name: SearchCandidates :many
WITH param AS (
    SELECT
        @search_date::date    AS search_date,
        @deadline_week::date  AS deadline_week,
        @quantity::int        AS quantity,
        sqlc.narg('product_item')::uuid  AS product_item,
        sqlc.narg('machine_item')::uuid  AS machine_item,
        sqlc.narg('max_lead')::int       AS max_lead,
        @searcher_profile::uuid          AS searcher_profile,
        sqlc.narg('city_code')::text     AS city_code,
        sqlc.narg('province_code')::text AS province_code
),
base_candidate AS (
    SELECT
        l.id AS listing_id,
        l.profile_id,
        pr.business_name,
        pr.verified,
        l.weekly_capacity,
        l.readiness_lead_days,
        l.horizon_until,
        date_trunc('week', p.search_date + (l.readiness_lead_days || ' days')::interval)::date
            AS readiness_week
    FROM capacity_listing l
    JOIN business_profile pr ON pr.id = l.profile_id
    CROSS JOIN param p
    WHERE l.published
      AND l.profile_id <> p.searcher_profile
      AND (p.city_code IS NULL OR pr.city_code = p.city_code)
      AND (p.province_code IS NULL OR pr.city_code IN (
              SELECT code FROM city WHERE province_code = p.province_code))
),
capacity AS (
    SELECT
        c.listing_id,
        coalesce(sum(pk.total_capacity - pk.used_capacity), 0) AS recorded_remaining,
        greatest(0, (
            (p.deadline_week - greatest(c.readiness_week, c.horizon_until + 7)) / 7 + 1
        )) * c.weekly_capacity AS uncreated_remaining
    FROM base_candidate c
    CROSS JOIN param p
    LEFT JOIN availability_period pk
           ON pk.listing_id = c.listing_id
          AND NOT pk.marked_full
          AND pk.week_start BETWEEN c.readiness_week AND p.deadline_week
    GROUP BY c.listing_id, c.readiness_week, c.horizon_until,
             c.weekly_capacity, p.deadline_week
),
scored AS (
    SELECT
        c.listing_id,
        c.profile_id,
        c.business_name,
        c.verified,
        c.readiness_lead_days,
        c.horizon_until,
        c.readiness_week,
        (cap.recorded_remaining + cap.uncreated_remaining)::bigint AS remaining_capacity,
        (p.product_item IS NULL OR EXISTS (
            SELECT 1 FROM listing_product lp
             WHERE lp.listing_id = c.listing_id AND lp.item_id = p.product_item))::int
            AS product_match,
        (p.machine_item IS NULL OR EXISTS (
            SELECT 1 FROM listing_machine lm
             WHERE lm.listing_id = c.listing_id AND lm.item_id = p.machine_item))::int
            AS machine_match,
        (p.max_lead IS NULL OR c.readiness_lead_days <= p.max_lead)::int
            AS lead_match,
        ((cap.recorded_remaining + cap.uncreated_remaining) >= p.quantity)::int
            AS capacity_enough
    FROM base_candidate c
    JOIN capacity cap ON cap.listing_id = c.listing_id
    CROSS JOIN param p
),
ranked AS (
    SELECT *,
           (product_match + machine_match + lead_match + capacity_enough) AS score
    FROM scored
)
SELECT
    listing_id,
    profile_id,
    business_name,
    verified,
    readiness_lead_days,
    horizon_until,
    remaining_capacity,
    product_match,
    machine_match,
    lead_match,
    capacity_enough,
    score
FROM ranked
WHERE (score, remaining_capacity, -readiness_lead_days, business_name, listing_id)
      < (@cursor_score::int, @cursor_remaining::bigint, @cursor_neg_lead::int,
         @cursor_name::text, @cursor_listing::uuid)
ORDER BY score DESC, remaining_capacity DESC, readiness_lead_days ASC,
         business_name ASC, listing_id ASC
LIMIT @page_size::int;
