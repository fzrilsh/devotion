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
-- (FR-061, driven by T068). It sets status, admin_note, decided_by, decided_at,
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

-- ListCatalogItemsByType returns every item of one type, active and inactive,
-- for the admin management surface (FR-059). Unlike ListActiveCatalogItems it
-- does not filter on active, so an admin sees deactivated items to reactivate or
-- rename them.
-- name: ListCatalogItemsByType :many
SELECT id, type, name, active FROM catalog_item
WHERE type = $1
ORDER BY sort_order, name;

-- InsertCatalogItem adds a new baseline item from the admin surface (FR-059) and
-- returns it. sort_order places new items after the seeded ones. It errors on the
-- item_name_unique_per_type constraint when a same-name item of the type exists,
-- which the handler maps to a validation error.
-- name: InsertCatalogItem :one
INSERT INTO catalog_item (id, type, name, active, sort_order, created_at)
VALUES (gen_random_uuid(), $1, $2, true, 1000, $3)
RETURNING id, type, name, active;

-- UpsertCatalogItemReturning resolves the catalog item an approved proposal
-- yields (FR-061): it inserts the proposed name, or reactivates and returns the
-- existing item when the same (type, name) is already present, and returns the
-- id either way. The decision path needs the id to satisfy approved_yields_item.
-- name: UpsertCatalogItemReturning :one
INSERT INTO catalog_item (id, type, name, active, sort_order, created_at)
VALUES (gen_random_uuid(), $1, $2, true, 1000, $3)
ON CONFLICT (type, name) DO UPDATE SET active = true
RETURNING id, type, name, active;

-- LockCatalogItem loads one item FOR UPDATE so a rename or deactivate reads and
-- writes the row under a row lock, matching the decision-path locking pattern.
-- name: LockCatalogItem :one
SELECT id, type, name, active FROM catalog_item
WHERE id = $1
FOR UPDATE;

-- UpdateCatalogItem applies a rename and/or an active flip to an existing item
-- (FR-059). COALESCE keeps the current value when the caller omits a field, so a
-- rename-only or deactivate-only PATCH leaves the other column untouched.
-- Deactivating only flips active; it never touches listing rows that reference
-- the item, so those listings stay discoverable (FR-060).
-- name: UpdateCatalogItem :one
UPDATE catalog_item
SET name = COALESCE(sqlc.narg('name'), name),
    active = COALESCE(sqlc.narg('active'), active)
WHERE id = $1
RETURNING id, type, name, active;

-- LockItemProposal loads one proposal FOR UPDATE, joined with the proposer's
-- account id, so the decision path reads the current status under a row lock
-- before deciding, matching the verification decision pattern.
-- name: LockItemProposal :one
SELECT p.*, bp.account_id AS proposer_account_id
FROM item_proposal p
JOIN business_profile bp ON bp.id = p.profile_id
WHERE p.id = $1
FOR UPDATE OF p;

-- ListItemProposalsPending returns the pending proposal queue oldest first, with
-- the proposer's business name, keyset paginated by (created_at, id) for a stable
-- opaque cursor (FR-061). Fetches limit+1 to detect a next page.
-- name: ListItemProposalsPending :many
SELECT p.id, p.type, p.proposed_name, p.status, p.admin_note, p.created_at,
       bp.business_name AS proposer_name
FROM item_proposal p
JOIN business_profile bp ON bp.id = p.profile_id
WHERE p.status = 'pending'
  AND (sqlc.narg('after_created_at')::timestamptz IS NULL
       OR (p.created_at, p.id) > (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid))
ORDER BY p.created_at, p.id
LIMIT $1;
