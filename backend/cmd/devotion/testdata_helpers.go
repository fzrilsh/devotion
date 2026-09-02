package main

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// nextPhone hands each seeded account a unique number matching the phone_format
// check (^62[0-9]{8,13}$). The 628xxxxxxxxx shape is a normal Indonesian mobile
// prefix; the numbers are synthetic and belong to no real subscriber (T075).
func (s *seeder) nextPhone() string {
	s.phoneSeq++
	return fmt.Sprintf("62811%07d", s.phoneSeq)
}

// demoHash returns the bcrypt digest of the shared demo password. Every seeded
// login uses it, so the manual test script can sign in as any fixture without a
// per-account secret. The commands refuse to run in production, so this never
// reaches a real database.
func (s *seeder) demoHash() (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// ensurePrerequisites makes sure the one region and the two catalog items the
// fixtures reference exist, then caches their ids on the seeder. It is
// idempotent: seed:regions and seed:master-data may already have run, so it
// upserts rather than assuming an empty database. The seeder needs a concrete
// city_code, product item, and machine item to attach every business and
// listing to.
func (s *seeder) ensurePrerequisites(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO province (code, name) VALUES ('32', 'Jawa Barat') ON CONFLICT (code) DO NOTHING`); err != nil {
		return fmt.Errorf("seed province: %w", err)
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO city (code, province_code, name) VALUES ('3273', '32', 'Kota Bandung') ON CONFLICT (code) DO NOTHING`); err != nil {
		return fmt.Errorf("seed city: %w", err)
	}
	s.cityCode = "3273"

	if err := s.pool.QueryRow(ctx,
		`INSERT INTO catalog_item (type, name, active, created_at)
		 VALUES ('product', 'Kaos Oblong', true, $1)
		 ON CONFLICT (type, name) DO UPDATE SET active = true
		 RETURNING id`, s.now).Scan(&s.productID); err != nil {
		return fmt.Errorf("seed product item: %w", err)
	}
	if err := s.pool.QueryRow(ctx,
		`INSERT INTO catalog_item (type, name, active, created_at)
		 VALUES ('machine', 'Mesin Jahit Jarum 1', true, $1)
		 ON CONFLICT (type, name) DO UPDATE SET active = true
		 RETURNING id`, s.now).Scan(&s.machineID); err != nil {
		return fmt.Errorf("seed machine item: %w", err)
	}
	return nil
}

// insertAccount inserts one .test account with the demo password and the given
// roles, returning its id. Every email uses testAccountDomain so reset can find
// the seeded rows and a demo login never collides with a real address.
func (s *seeder) insertAccount(ctx context.Context, localPart string, subcontractor, buyer bool) (string, error) {
	hash, err := s.demoHash()
	if err != nil {
		return "", err
	}
	email := localPart + "@" + testAccountDomain
	var id string
	err = s.pool.QueryRow(ctx,
		`INSERT INTO user_account
		   (email, phone, password_hash, email_verified, phone_verified,
		    role_subcontractor, role_buyer, created_at, updated_at)
		 VALUES ($1, $2, $3, true, true, $4, $5, $6, $6)
		 RETURNING id`,
		email, s.nextPhone(), hash, subcontractor, buyer, s.now).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("seed account %s: %w", email, err)
	}
	return id, nil
}

// insertProfile inserts a verified business profile for an account and returns
// its id. Coordinates are placed inside Bandung so the informational distance in
// search results has something to compute.
func (s *seeder) insertProfile(ctx context.Context, accountID, name string, verified bool) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO business_profile
		   (account_id, business_name, city_code, latitude, longitude, verified, created_at, updated_at)
		 VALUES ($1, $2, $3, -6.914744, 107.609810, $4, $5, $5)
		 RETURNING id`,
		accountID, name, s.cityCode, verified, s.now).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("seed profile %s: %w", name, err)
	}
	return id, nil
}

// newSubcontractor creates a subcontractor account and its verified profile in
// one step, the common shape for every listing owner.
func (s *seeder) newSubcontractor(ctx context.Context, slug, name string) (profileID string, err error) {
	acc, err := s.insertAccount(ctx, slug, true, false)
	if err != nil {
		return "", err
	}
	return s.insertProfile(ctx, acc, name, true)
}

// newBuyer creates a buyer account and its verified profile in one step.
func (s *seeder) newBuyer(ctx context.Context, slug, name string) (profileID string, err error) {
	acc, err := s.insertAccount(ctx, slug, false, true)
	if err != nil {
		return "", err
	}
	return s.insertProfile(ctx, acc, name, true)
}

// insertListing creates a published capacity listing for the product item and
// materializes availability periods from this week through horizonWeeks weeks
// ahead, each at total_capacity = weeklyCap. calendarUpdatedAt controls the
// FR-021 staleness clock: pass s.now for a fresh calendar, or an older instant
// to seed a stale one. horizonWeeks must be at least 0; the horizon_until column
// is the Monday of the last materialized week.
func (s *seeder) insertListing(ctx context.Context, profileID string, weeklyCap, leadDays, horizonWeeks int, calendarUpdatedAt any) (string, error) {
	horizonUntil := s.thisWeek.AddDate(0, 0, 7*horizonWeeks)
	var listingID string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO capacity_listing
		   (profile_id, weekly_capacity, readiness_lead_days, published,
		    calendar_updated_at, horizon_until, created_at, updated_at)
		 VALUES ($1, $2, $3, true, $4, $5, $6, $6)
		 RETURNING id`,
		profileID, weeklyCap, leadDays, calendarUpdatedAt, horizonUntil, s.now).Scan(&listingID)
	if err != nil {
		return "", fmt.Errorf("seed listing: %w", err)
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO listing_product (listing_id, item_id) VALUES ($1, $2)`,
		listingID, s.productID); err != nil {
		return "", fmt.Errorf("seed listing_product: %w", err)
	}
	for w := 0; w <= horizonWeeks; w++ {
		week := s.thisWeek.AddDate(0, 0, 7*w)
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO availability_period
			   (listing_id, week_start, total_capacity, used_capacity, created_at, updated_at)
			 VALUES ($1, $2, $3, 0, $4, $4)`,
			listingID, week, weeklyCap, s.now); err != nil {
			return "", fmt.Errorf("seed period %s: %w", week.Format("2006-01-02"), err)
		}
	}
	return listingID, nil
}

// orderSpec describes one work order the seeder builds end to end, from the
// buyer's quota request through the agreed candidate, the accepted offer, the
// work order itself, and the capacity allocation that draws down the
// subcontractor's period. Timestamps are passed in already resolved from the
// clock so a deadline-bound row lands on a known side of its window.
type orderSpec struct {
	slug          string
	buyerName     string
	subconName    string
	quantity      int
	weeklyCap     int
	totalPrice    int64
	deadline      time.Time // date column, truncated to the day
	readinessWeek time.Time // Monday, must be <= deadline
	status        string
	shippedAt     *time.Time // nil unless the order has reached "shipped"
	createdAt     time.Time
}

// insertOrder materializes the full order chain for one spec and returns the new
// work_order id. It creates a dedicated buyer and subcontractor so the order
// stands on its own two parties (the two_distinct_parties check), publishes a
// listing for the subcontractor with a single availability period on the
// readiness week, then writes the request, the agreed candidate, the accepted
// offer, the work order, and the allocation drawing quantity from that period.
// The allocation's period is the readiness week itself, so the FR-087
// before-readiness trigger is satisfied.
func (s *seeder) insertOrder(ctx context.Context, spec orderSpec) (string, error) {
	buyer, err := s.newBuyer(ctx, spec.slug+"-pembeli", spec.buyerName)
	if err != nil {
		return "", err
	}
	subcon, err := s.newSubcontractor(ctx, spec.slug+"-penjahit", spec.subconName)
	if err != nil {
		return "", err
	}

	listingID, err := s.insertOrderListing(ctx, subcon, spec.weeklyCap, spec.readinessWeek)
	if err != nil {
		return "", err
	}

	return s.insertOrderOnListing(ctx, orderOnListing{
		slug:          spec.slug,
		listingID:     listingID,
		buyerID:       buyer,
		subcontractor: subcon,
		quantity:      spec.quantity,
		totalPrice:    spec.totalPrice,
		deadline:      spec.deadline,
		readinessWeek: spec.readinessWeek,
		status:        spec.status,
		shippedAt:     spec.shippedAt,
		createdAt:     spec.createdAt,
	})
}

// orderOnListing writes one order chain against a listing that already exists,
// so several orders can share one subcontractor and its completion divisor. The
// buyer and subcontractor profiles are passed in rather than minted here.
type orderOnListing struct {
	slug          string
	listingID     string
	buyerID       string
	subcontractor string
	quantity      int
	totalPrice    int64
	deadline      time.Time
	readinessWeek time.Time
	status        string
	shippedAt     *time.Time
	createdAt     time.Time
}

// insertOrderOnListing writes the request, agreed candidate, accepted offer,
// work order, and capacity allocation for one order against an existing listing,
// then draws the quantity down from the readiness-week period. It returns the
// new work_order id.
func (s *seeder) insertOrderOnListing(ctx context.Context, o orderOnListing) (string, error) {
	var requestID string
	if err := s.pool.QueryRow(ctx,
		`INSERT INTO quota_request
		   (buyer_id, product_item_id, quantity, material, deadline, reply_due_at, created_at)
		 VALUES ($1, $2, $3, 'Katun Combed 30s', $4, $5, $6)
		 RETURNING id`,
		o.buyerID, s.productID, o.quantity, o.deadline,
		o.createdAt.Add(72*time.Hour), o.createdAt).Scan(&requestID); err != nil {
		return "", fmt.Errorf("seed request for %s: %w", o.slug, err)
	}

	var candidateID string
	if err := s.pool.QueryRow(ctx,
		`INSERT INTO request_candidate
		   (request_id, listing_id, subcontractor_id, status, updated_at)
		 VALUES ($1, $2, $3, 'agreed', $4)
		 RETURNING id`,
		requestID, o.listingID, o.subcontractor, o.createdAt).Scan(&candidateID); err != nil {
		return "", fmt.Errorf("seed candidate for %s: %w", o.slug, err)
	}

	var offerID string
	if err := s.pool.QueryRow(ctx,
		`INSERT INTO offer
		   (candidate_id, sequence, proposed_by, total_price, readiness_lead_days, created_at)
		 VALUES ($1, 1, 'subcontractor', $2, 0, $3)
		 RETURNING id`,
		candidateID, o.totalPrice, o.createdAt).Scan(&offerID); err != nil {
		return "", fmt.Errorf("seed offer for %s: %w", o.slug, err)
	}

	var orderID string
	if err := s.pool.QueryRow(ctx,
		`INSERT INTO work_order
		   (candidate_id, offer_id, buyer_id, subcontractor_id, quantity, total_price,
		    deadline, readiness_week_start, status, shipped_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id`,
		candidateID, offerID, o.buyerID, o.subcontractor, o.quantity, o.totalPrice,
		o.deadline, o.readinessWeek, o.status, o.shippedAt, o.createdAt).Scan(&orderID); err != nil {
		return "", fmt.Errorf("seed work order for %s: %w", o.slug, err)
	}

	if _, err := s.pool.Exec(ctx,
		`INSERT INTO capacity_allocation (work_order_id, period_id, quantity, created_at)
		 SELECT $1, p.id, $2, $3
		   FROM availability_period p
		  WHERE p.listing_id = $4 AND p.week_start = $5`,
		orderID, o.quantity, o.createdAt, o.listingID, o.readinessWeek); err != nil {
		return "", fmt.Errorf("seed allocation for %s: %w", o.slug, err)
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE availability_period SET used_capacity = used_capacity + $1, updated_at = $2
		  WHERE listing_id = $3 AND week_start = $4`,
		o.quantity, o.createdAt, o.listingID, o.readinessWeek); err != nil {
		return "", fmt.Errorf("seed period drawdown for %s: %w", o.slug, err)
	}
	return orderID, nil
}

// insertUploadedFile inserts one uploaded_file row for a profile and returns its
// id. The bytes are synthetic; only the row is needed so a verification_request
// can reference an identity document and a location photo. storage_path is made
// unique per file so the storage_path_unique constraint holds across the fixture
// set.
func (s *seeder) insertUploadedFile(ctx context.Context, profileID, fileType, originalName, mimeType string) (string, error) {
	s.fileSeq++
	storagePath := fmt.Sprintf("seed/%s/%06d", fileType, s.fileSeq)
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO uploaded_file
		   (owner_profile_id, type, original_name, mime_type, size_bytes, storage_path, created_at)
		 VALUES ($1, $2, $3, $4, 1024, $5, $6)
		 RETURNING id`,
		profileID, fileType, originalName, mimeType, storagePath, s.now).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("seed uploaded file %s: %w", originalName, err)
	}
	return id, nil
}


// insertOrderListing publishes a listing whose one availability period sits on
// the order's readiness week, big enough to hold the allocation. The horizon is
// the readiness week itself, so horizon_until stays a Monday.
func (s *seeder) insertOrderListing(ctx context.Context, profileID string, weeklyCap int, readinessWeek time.Time) (string, error) {
	var listingID string
	if err := s.pool.QueryRow(ctx,
		`INSERT INTO capacity_listing
		   (profile_id, weekly_capacity, readiness_lead_days, published,
		    calendar_updated_at, horizon_until, created_at, updated_at)
		 VALUES ($1, $2, 0, true, $3, $4, $3, $3)
		 RETURNING id`,
		profileID, weeklyCap, s.now, readinessWeek).Scan(&listingID); err != nil {
		return "", fmt.Errorf("seed order listing: %w", err)
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO listing_product (listing_id, item_id) VALUES ($1, $2)`,
		listingID, s.productID); err != nil {
		return "", fmt.Errorf("seed order listing_product: %w", err)
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO availability_period
		   (listing_id, week_start, total_capacity, used_capacity, created_at, updated_at)
		 VALUES ($1, $2, $3, 0, $4, $4)`,
		listingID, readinessWeek, weeklyCap, s.now); err != nil {
		return "", fmt.Errorf("seed order period: %w", err)
	}
	return listingID, nil
}

