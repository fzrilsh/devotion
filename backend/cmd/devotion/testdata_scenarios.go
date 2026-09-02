package main

import (
	"context"
	"fmt"
	"time"
)

// seedGenericBusinesses creates the filler pool: genericBusinessCount
// subcontractors, each with a published listing carrying the product item, so
// the search result set spans more than one page and the keyset pagination
// scenario has something to page through. They are deliberately uniform; the
// showcase listings below carry the distinguishing capacities and leads.
func (s *seeder) seedGenericBusinesses(ctx context.Context, res *seedResult) error {
	for i := 1; i <= genericBusinessCount; i++ {
		profile, err := s.newSubcontractor(ctx,
			fmt.Sprintf("umkm-%02d", i),
			fmt.Sprintf("Konveksi Contoh %02d", i))
		if err != nil {
			return err
		}
		if _, err := s.insertListing(ctx, profile, 100, 7, 12, s.now); err != nil {
			return err
		}
		res.Businesses++
		res.Listings++
	}
	return nil
}

// seedShowcaseListings creates the four listings the matching and readiness
// scenarios name explicitly:
//
//   - 500 pieces/week, 0-day lag: the 3.000-piece order fills across six weeks
//     starting this week, the FR-088 multi-week fill demo.
//   - 14-day and 21-day lag: their readiness weeks land two and three Mondays
//     out, so the readiness-week rule (FR-087) is visible side by side.
//   - 200 pieces/week: a five-month deadline is needed to fit a large order, the
//     long-horizon scenario.
//
// Each is owned by its own subcontractor so they show up as distinct candidates.
func (s *seeder) seedShowcaseListings(ctx context.Context, res *seedResult) error {
	type showcase struct {
		slug, name             string
		weeklyCap, lead, weeks int
	}
	cases := []showcase{
		{"kapasitas-500", "Konveksi Kapasitas 500", 500, 0, 12},
		{"jeda-14", "Konveksi Jeda 14 Hari", 300, 14, 12},
		{"jeda-21", "Konveksi Jeda 21 Hari", 300, 21, 12},
		{"kapasitas-200", "Konveksi Kapasitas 200", 200, 0, 26},
	}
	for _, c := range cases {
		profile, err := s.newSubcontractor(ctx, c.slug, c.name)
		if err != nil {
			return err
		}
		if _, err := s.insertListing(ctx, profile, c.weeklyCap, c.lead, c.weeks, s.now); err != nil {
			return err
		}
		res.Businesses++
		res.Listings++
	}
	return nil
}

// seedStaleCalendar creates a listing whose calendar has not been touched for
// eight days, one day past the FR-021 seven-day staleness window, so the "belum
// diperbarui" surfacing has a clear-cut example. The staleness is seeded as an
// old calendar_updated_at value, not by advancing a clock, so it cannot drift
// (T075).
func (s *seeder) seedStaleCalendar(ctx context.Context, res *seedResult) error {
	profile, err := s.newSubcontractor(ctx, "kalender-basi", "Konveksi Kalender Basi")
	if err != nil {
		return err
	}
	staleAt := s.now.Add(-8 * 24 * time.Hour)
	if _, err := s.insertListing(ctx, profile, 150, 7, 12, staleAt); err != nil {
		return err
	}
	res.Businesses++
	res.Listings++
	return nil
}

// seedExpiredRequest creates a buyer with a quota request whose 72-hour reply
// window has already closed (reply_due_at three days before now), the expired
// state for the request-expiry scenario. It carries no candidate, so it stands
// as a request that lapsed without a reply. The expiry is seeded as a past
// reply_due_at, never by moving a clock (T075).
func (s *seeder) seedExpiredRequest(ctx context.Context, res *seedResult) error {
	buyer, err := s.newBuyer(ctx, "pembeli-kedaluwarsa", "Pemesan Kedaluwarsa")
	if err != nil {
		return err
	}
	res.Businesses++

	createdAt := s.now.Add(-4 * 24 * time.Hour)
	replyDueAt := s.now.Add(-24 * time.Hour)
	deadline := s.thisWeek.AddDate(0, 0, 7*8)
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO quota_request
		   (buyer_id, product_item_id, quantity, material, deadline, reply_due_at, created_at)
		 VALUES ($1, $2, 1000, 'Katun Combed 30s', $3, $4, $5)`,
		buyer, s.productID, deadline, replyDueAt, createdAt); err != nil {
		return fmt.Errorf("seed expired request: %w", err)
	}
	return nil
}

// seedShippedAndLateOrders builds the three deadline-bound orders the
// auto-confirm and late-order scenarios name. Each is seeded already sitting in
// its target state, never by moving a clock (T075):
//
//   - shipped 6 days ago: shipped_at is Clock.Now() minus 6 days, so the 7-day
//     auto-confirm window (FR-068) has not closed yet and the buyer can still
//     confirm or dispute.
//   - shipped 8 days ago: shipped_at is Clock.Now() minus 8 days, past the 7-day
//     window, so the auto-confirm job has an order to close on its next tick.
//   - late: an active order whose deadline fell before today's WIB day
//     (FR-045), so it shows on the admin late list and the notifier scan.
func (s *seeder) seedShippedAndLateOrders(ctx context.Context, res *seedResult) error {
	shipped6 := s.now.AddDate(0, 0, -6)
	shipped8 := s.now.AddDate(0, 0, -8)

	specs := []orderSpec{
		{
			slug:          "dikirim-6-hari",
			buyerName:     "Pemesan Dikirim 6 Hari",
			subconName:    "Penjahit Dikirim 6 Hari",
			quantity:      120,
			weeklyCap:     200,
			totalPrice:    3_600_000,
			deadline:      s.thisWeek.AddDate(0, 0, 14),
			readinessWeek: s.thisWeek,
			status:        "shipped",
			shippedAt:     &shipped6,
			createdAt:     s.now.AddDate(0, 0, -20),
		},
		{
			slug:          "dikirim-8-hari",
			buyerName:     "Pemesan Dikirim 8 Hari",
			subconName:    "Penjahit Dikirim 8 Hari",
			quantity:      150,
			weeklyCap:     200,
			totalPrice:    4_500_000,
			deadline:      s.thisWeek.AddDate(0, 0, 7),
			readinessWeek: s.thisWeek,
			status:        "shipped",
			shippedAt:     &shipped8,
			createdAt:     s.now.AddDate(0, 0, -25),
		},
		{
			slug:          "pesanan-telat",
			buyerName:     "Pemesan Telat",
			subconName:    "Penjahit Telat",
			quantity:      100,
			weeklyCap:     200,
			totalPrice:    3_000_000,
			deadline:      s.thisWeek.AddDate(0, 0, -7),
			readinessWeek: s.thisWeek.AddDate(0, 0, -14),
			status:        "production",
			shippedAt:     nil,
			createdAt:     s.now.AddDate(0, 0, -30),
		},
	}
	for _, spec := range specs {
		if _, err := s.insertOrder(ctx, spec); err != nil {
			return err
		}
		res.Businesses += 2
		res.Listings++
		res.Orders++
	}
	return nil
}

// seedVerifications creates two profiles each with a pending verification
// submission, the moderation queue the admin verification scenario works
// through. Each submission needs an identity document and a location photo
// uploaded first (uploaded_file), then the verification_request that references
// them. The identity numbers are synthetic 16-digit strings that belong to no
// real person (T075).
func (s *seeder) seedVerifications(ctx context.Context, res *seedResult) error {
	subjects := []struct {
		slug, name, identity string
	}{
		{"verifikasi-1", "Konveksi Menunggu Verifikasi 1", "3273010101010001"},
		{"verifikasi-2", "Konveksi Menunggu Verifikasi 2", "3273020202020002"},
	}
	for _, subj := range subjects {
		profile, err := s.newSubcontractor(ctx, subj.slug, subj.name)
		if err != nil {
			return err
		}
		res.Businesses++

		identityFile, err := s.insertUploadedFile(ctx, profile, "identity_document",
			"ktp-"+subj.slug+".jpg", "image/jpeg")
		if err != nil {
			return err
		}
		locationFile, err := s.insertUploadedFile(ctx, profile, "location_photo",
			"lokasi-"+subj.slug+".jpg", "image/jpeg")
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO verification_request
			   (profile_id, identity_number, identity_file_id, location_file_id, status, created_at)
			 VALUES ($1, $2, $3, $4, 'pending', $5)`,
			profile, subj.identity, identityFile, locationFile, s.now); err != nil {
			return fmt.Errorf("seed verification %s: %w", subj.slug, err)
		}
		res.Verifications++
	}
	return nil
}

// seedFewOrdersBusiness creates a subcontractor with exactly two confirmed
// orders, one short of the FR-073 threshold of three, so the completion rate is
// withheld and the "belum cukup data" note shows instead of a percentage. Both
// orders are already confirmed, so nothing about them is deadline-bound. The two
// orders share one subcontractor, so its completion divisor is 2; each gets its
// own buyer to satisfy two_distinct_parties.
func (s *seeder) seedFewOrdersBusiness(ctx context.Context, res *seedResult) error {
	subcon, err := s.newSubcontractor(ctx, "dua-pesanan-penjahit", "Penjahit Dua Pesanan")
	if err != nil {
		return err
	}
	res.Businesses++

	readinessWeek := s.thisWeek.AddDate(0, 0, -28)
	listingID, err := s.insertOrderListing(ctx, subcon, 400, readinessWeek)
	if err != nil {
		return err
	}
	res.Listings++

	confirmedAt := s.now.AddDate(0, 0, -40)
	for i := 1; i <= 2; i++ {
		buyer, err := s.newBuyer(ctx,
			fmt.Sprintf("dua-pesanan-%d-pembeli", i),
			fmt.Sprintf("Pemesan Dua Pesanan %d", i))
		if err != nil {
			return err
		}
		res.Businesses++
		if _, err := s.insertOrderOnListing(ctx, orderOnListing{
			slug:          fmt.Sprintf("dua-pesanan-%d", i),
			listingID:     listingID,
			buyerID:       buyer,
			subcontractor: subcon,
			quantity:      80,
			totalPrice:    2_400_000,
			deadline:      s.thisWeek.AddDate(0, 0, -21),
			readinessWeek: readinessWeek,
			status:        "confirmed",
			createdAt:     confirmedAt,
		}); err != nil {
			return err
		}
		res.Orders++
	}
	return nil
}


