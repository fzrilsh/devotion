-- Master data queries: reference regions (province, city) and the baseline
-- catalog of product and machine types. The seeders upsert by code/name so
-- running them twice never duplicates and never deletes; business_profile
-- references these rows, so a delete would orphan real data.

-- UpsertProvince inserts a province or updates its name when the code already
-- exists. Idempotent on code: the seed can run repeatedly without duplicating.
-- name: UpsertProvince :exec
INSERT INTO province (code, name)
VALUES ($1, $2)
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name;

-- UpsertCity inserts a city or updates its name. The code is already normalized
-- (dots stripped, four digits) by the seeder before it reaches here, so the
-- city_code_format and city_belongs_to_province constraints hold.
-- name: UpsertCity :exec
INSERT INTO city (code, province_code, name)
VALUES ($1, $2, $3)
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, province_code = EXCLUDED.province_code;

-- ListProvinces returns every province ordered by code, for GET /regions/provinces.
-- name: ListProvinces :many
SELECT code, name FROM province ORDER BY code;

-- ListCities returns every city ordered by code, for GET /regions/cities with no
-- province filter.
-- name: ListCities :many
SELECT code, province_code, name FROM city ORDER BY code;

-- ListCitiesByProvince returns the cities of one province, for
-- GET /regions/cities?province=.
-- name: ListCitiesByProvince :many
SELECT code, province_code, name FROM city WHERE province_code = $1 ORDER BY code;

-- UpsertCatalogItem inserts a baseline product or machine type, or reactivates
-- and reorders an existing one. Idempotent on (type, name): the seed can run
-- twice without duplicating. created_at is supplied by the caller from the Clock
-- since the column has no DB default.
-- name: UpsertCatalogItem :exec
INSERT INTO catalog_item (id, type, name, active, sort_order, created_at)
VALUES (gen_random_uuid(), $1, $2, true, $3, $4)
ON CONFLICT (type, name) DO UPDATE SET active = true, sort_order = EXCLUDED.sort_order;

-- ListActiveCatalogItems returns the active items of one type ordered for
-- display, for GET /master/products and /master/machines.
-- name: ListActiveCatalogItems :many
SELECT id, type, name, active FROM catalog_item
WHERE type = $1 AND active
ORDER BY sort_order, name;

-- InsertItemProposal records a user's proposal for a new catalog item (FR-061).
-- profile_id is the proposer's business profile, type is product or machine, and
-- created_at comes from the Clock since the column has no DB default. status
-- defaults to 'pending'; decided_* stay null until an admin decides.
-- name: InsertItemProposal :one
INSERT INTO item_proposal (profile_id, type, proposed_name, created_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- GetItemProposalByID loads one proposal, joined with the proposer's account id
-- so the decision path knows whom to notify. account_id is the business
-- profile's owner, the recipient of the item_proposal_decision notification.
-- name: GetItemProposalByID :one
SELECT p.*, bp.account_id AS proposer_account_id
FROM item_proposal p
JOIN business_profile bp ON bp.id = p.profile_id
WHERE p.id = $1;

-- DecideItemProposal applies an admin decision to a still-pending proposal
-- (FR-058, driven by T068). It sets status, admin_note, decided_by, decided_at,
-- and the resulting item_id (non-null only on approval), guarding on the current
-- 'pending' status so a second decision on the same row affects nothing and
-- RETURNING yields no row. The decision_complete and approved_yields_item table
-- constraints enforce the shape of an approved versus rejected decision.
-- name: DecideItemProposal :one
UPDATE item_proposal
SET status = $2, admin_note = $3, decided_by = $4, decided_at = $5, item_id = $6
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- CountProvinces, CountCities, CountCatalogByType back the seed verification and
-- the "count is greater than zero" done-check.
-- name: CountProvinces :one
SELECT count(*)::bigint FROM province;

-- name: CountCities :one
SELECT count(*)::bigint FROM city;

-- name: CountCatalogByType :one
SELECT count(*)::bigint FROM catalog_item WHERE type = $1;
