-- CreateProfile inserts the business profile that is born together with the
-- account inside the registration transaction. latitude and longitude arrive
-- null at registration; the owner sets them later through PUT /profile/me. The
-- coordinates_complete_or_empty and coordinates_within_indonesia constraints
-- accept a null pair, so a fresh profile carries no location yet.
-- name: CreateProfile :one
INSERT INTO business_profile (
    id, account_id, business_name, city_code,
    created_at, updated_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $4
)
RETURNING *;

-- GetProfileByAccount loads the caller's own profile by account id, backing
-- GET /profile/me. The city join resolves the human-readable city and province
-- names the contract returns as read-only fields, so the client never has to
-- look them up separately.
-- name: GetProfileByAccount :one
SELECT p.*, c.name AS city_name, c.province_code, pr.name AS province_name
FROM business_profile p
JOIN city c ON c.code = p.city_code
JOIN province pr ON pr.code = c.province_code
WHERE p.account_id = $1;

-- GetProfileByID loads any profile by its id for the public view, joining city
-- and province for their names. The public handler decides which columns to
-- expose; this query returns the full row and the caller projects it.
-- name: GetProfileByID :one
SELECT p.*, c.name AS city_name, c.province_code, pr.name AS province_name
FROM business_profile p
JOIN city c ON c.code = p.city_code
JOIN province pr ON pr.code = c.province_code
WHERE p.id = $1;

-- UpdateProfile writes the fields the owner may change through PUT /profile/me:
-- the business name, the chosen city, the map coordinates, and the free-text
-- description. account_id and verified are never touched here.
-- name: UpdateProfile :one
UPDATE business_profile
SET business_name = $2,
    city_code     = $3,
    latitude      = $4,
    longitude     = $5,
    description   = $6,
    updated_at    = $7
WHERE id = $1
RETURNING *;

-- CityExists reports whether a city code is known, so registration and profile
-- edits can answer 422 on an unknown city instead of surfacing a foreign key
-- violation as a 500.
-- name: CityExists :one
SELECT EXISTS (SELECT 1 FROM city WHERE code = $1);
