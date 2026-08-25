package order

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/listing"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// acceptBaseTime is a fixed Monday (2026-08-24) so WeekStart(now) is the day
// itself and the readiness/deadline weeks are deterministic.
var acceptBaseTime = time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)

// acceptPhoneSeq hands each seeded account a unique phone matching the
// phone_format check (^62[0-9]{8,13}$).
var acceptPhoneSeq atomic.Int64

func acceptPhone() string {
	return "628" + strconv.Itoa(200000000+int(acceptPhoneSeq.Add(1)))
}

// noopNotifier satisfies Notifier without a queue; the accept tests assert on
// capacity and error codes, not on notification rows.
type noopNotifier struct{}

func (noopNotifier) Enqueue(_ context.Context, _ pgx.Tx, _ pgtype.UUID, _ sqlcgen.EventType, _, _ string, _ *string) error {
	return nil
}

// acceptHarness owns a migrated schema, the real listing service as the
// HorizonEnsurer, and the order service under test driven by a TestClock.
type acceptHarness struct {
	svc       *Service
	pool      *pgxpool.Pool
	clock     *platform.TestClock
	productID pgtype.UUID
}

func newAcceptHarness(t *testing.T, name string) *acceptHarness {
	t.Helper()
	pool := testdb.New(t, name)
	clock := platform.NewTestClock(acceptBaseTime)
	ls := listing.New(pool, clock)
	svc := New(pool, clock, noopNotifier{}, ls)

	seedAcceptRegion(t, pool)
	productID := seedAcceptCatalogItem(t, pool, "product", "Kaos Oblong")

	return &acceptHarness{svc: svc, pool: pool, clock: clock, productID: productID}
}

func seedAcceptRegion(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO province (code, name) VALUES ('32', 'Jawa Barat') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seed province: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO city (code, province_code, name) VALUES ('3273', '32', 'Kota Bandung') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seed city: %v", err)
	}
}

func seedAcceptCatalogItem(t *testing.T, pool *pgxpool.Pool, kind sqlcgen.ItemType, name string) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO catalog_item (type, name, active, created_at) VALUES ($1, $2, true, $3) RETURNING id`,
		kind, name, acceptBaseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed catalog item: %v", err)
	}
	return id
}

func seedAcceptAccount(t *testing.T, pool *pgxpool.Pool, email string, subcontractor bool) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO user_account (email, phone, password_hash, role_subcontractor, role_buyer, created_at, updated_at)
		 VALUES ($1, $2, 'x', $3, $4, $5, $5) RETURNING id`,
		email, acceptPhone(), subcontractor, !subcontractor, acceptBaseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}
	return id
}

func seedAcceptProfile(t *testing.T, pool *pgxpool.Pool, accountID pgtype.UUID, name string) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO business_profile (account_id, business_name, city_code, verified, created_at, updated_at)
		 VALUES ($1, $2, '3273', true, $3, $3) RETURNING id`,
		accountID, name, acceptBaseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	return id
}

// seedListing creates a published listing owned by a fresh subcontractor and
// materializes availability periods from the base week up to (and including)
// periodsUntil. weeklyCap is both the listing's weekly_capacity and each
// period's total_capacity. horizonUntil is stored on the listing so a test can
// place the deadline beyond it and exercise EnsureHorizon.
func seedListing(t *testing.T, h *acceptHarness, name string, weeklyCap int, horizonUntil, periodsUntil time.Time) (listingID, subAccount pgtype.UUID) {
	t.Helper()
	ctx := context.Background()
	pool := h.pool

	subAccount = seedAcceptAccount(t, pool, "sub-"+name+"@contoh.test", true)
	prof := seedAcceptProfile(t, pool, subAccount, "Subkon "+name)

	err := pool.QueryRow(ctx,
		`INSERT INTO capacity_listing (profile_id, weekly_capacity, readiness_lead_days, published, calendar_updated_at, horizon_until, created_at, updated_at)
		 VALUES ($1, $2, 0, true, $3, $4, $3, $3) RETURNING id`,
		prof, weeklyCap, acceptBaseTime, horizonUntil).Scan(&listingID)
	if err != nil {
		t.Fatalf("seed listing: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO listing_product (listing_id, item_id) VALUES ($1, $2)`,
		listingID, h.productID); err != nil {
		t.Fatalf("seed listing_product: %v", err)
	}
	base := platform.WeekStart(acceptBaseTime)
	for w := base; !w.After(periodsUntil); w = w.AddDate(0, 0, 7) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO availability_period (listing_id, week_start, total_capacity, used_capacity, created_at, updated_at)
			 VALUES ($1, $2, $3, 0, $4, $4)`,
			listingID, w, weeklyCap, acceptBaseTime); err != nil {
			t.Fatalf("seed period %s: %v", w.Format("2006-01-02"), err)
		}
	}
	return listingID, subAccount
}

// seedRequest inserts a quota request for the buyer with the given quantity and
// deadline week, returning its id.
func seedRequest(t *testing.T, h *acceptHarness, buyerProfile pgtype.UUID, quantity int, deadline time.Time) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	err := h.pool.QueryRow(context.Background(),
		`INSERT INTO quota_request (buyer_id, product_item_id, quantity, material, deadline, reply_due_at, created_at)
		 VALUES ($1, $2, $3, 'Katun', $4, $5, $6) RETURNING id`,
		buyerProfile, h.productID, quantity, deadline,
		acceptBaseTime.Add(72*time.Hour), acceptBaseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed request: %v", err)
	}
	return id
}

// seedOfferedCandidate wires a candidate in status 'offered' on the listing with
// a single standing offer, the shape GetOfferForAccept requires. It returns the
// offer id the buyer accepts.
func seedOfferedCandidate(t *testing.T, h *acceptHarness, requestID, listingID, subProfile pgtype.UUID, price int64) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	var candidateID pgtype.UUID
	err := h.pool.QueryRow(ctx,
		`INSERT INTO request_candidate (request_id, listing_id, subcontractor_id, status, updated_at)
		 VALUES ($1, $2, $3, 'offered', $4) RETURNING id`,
		requestID, listingID, subProfile, acceptBaseTime).Scan(&candidateID)
	if err != nil {
		t.Fatalf("seed candidate: %v", err)
	}
	var offerID pgtype.UUID
	err = h.pool.QueryRow(ctx,
		`INSERT INTO offer (candidate_id, sequence, proposed_by, total_price, readiness_lead_days, created_at)
		 VALUES ($1, 1, 'subcontractor', $2, 0, $3) RETURNING id`,
		candidateID, price, acceptBaseTime).Scan(&offerID)
	if err != nil {
		t.Fatalf("seed offer: %v", err)
	}
	return offerID
}

// subProfileID looks up the business_profile id owned by an account, so a test
// can point a second request's candidate at the same listing owner.
func subProfileID(t *testing.T, h *acceptHarness, account pgtype.UUID) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`SELECT id FROM business_profile WHERE account_id = $1`, account).Scan(&id); err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	return id
}

// errCode pulls the stable machine code out of either service error shape.
func errCode(err error) httpx.Code {
	var c *conflictError
	if errors.As(err, &c) {
		return c.code
	}
	var m *metaError
	if errors.As(err, &m) {
		return m.code
	}
	return ""
}

// usedCapacity sums used_capacity across a listing's periods.
func usedCapacity(t *testing.T, h *acceptHarness, listingID pgtype.UUID) int64 {
	t.Helper()
	var sum int64
	if err := h.pool.QueryRow(context.Background(),
		`SELECT coalesce(sum(used_capacity), 0) FROM availability_period WHERE listing_id = $1`,
		listingID).Scan(&sum); err != nil {
		t.Fatalf("sum used_capacity: %v", err)
	}
	return sum
}

// TestAccept_ConcurrentDifferentRequests_LoserGetsCapacityAlreadyTaken_FR036 runs
// two accepts against the SAME period from two DIFFERENT requests. Capacity fits
// exactly one order, so one commits and the other, having passed the pre-lock
// estimate, comes up short only under the lock: it must get CAPACITY_ALREADY_TAKEN,
// not INSUFFICIENT_CAPACITY. The used_capacity ending at one order's quantity, not
// two, is what proves FR-036 held, beyond the error code alone. The listing's
// horizon reaches the deadline so UncreatedRemaining is zero and the only cause of
// a post-lock shortfall is the real race.
func TestAccept_ConcurrentDifferentRequests_LoserGetsCapacityAlreadyTaken_FR036(t *testing.T) {
	h := newAcceptHarness(t, "accept_race_capacity")
	week := platform.WeekStart(acceptBaseTime)

	// One week only, total capacity 60: one order of 50 fits, two do not.
	listingID, subAcc := seedListing(t, h, "alfa", 60, week, week)
	subProf := subProfileID(t, h, subAcc)

	buyerA := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyerA@contoh.test", false), "Pembeli A")
	buyerB := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyerB@contoh.test", false), "Pembeli B")

	reqA := seedRequest(t, h, buyerA, 50, week)
	reqB := seedRequest(t, h, buyerB, 50, week)
	offerA := seedOfferedCandidate(t, h, reqA, listingID, subProf, 1_000_000)
	offerB := seedOfferedCandidate(t, h, reqB, listingID, subProf, 1_000_000)

	accA := buyerAccountOf(t, h, buyerA)
	accB := buyerAccountOf(t, h, buyerB)

	errs := runConcurrent(
		func() error { _, err := h.svc.accept(context.Background(), accA, offerA); return err },
		func() error { _, err := h.svc.accept(context.Background(), accB, offerB); return err },
	)

	assertOneWinnerOneCode(t, errs, httpx.CodeCapacityAlreadyTaken)

	if got := usedCapacity(t, h, listingID); got != 50 {
		t.Fatalf("used_capacity total = %d, mau 50 (satu pesanan, bukan dua): FR-036 gagal", got)
	}
}

// TestAccept_ConcurrentSameRequest_LoserGetsRequestAlreadyAgreed_FR034 runs two
// accepts on the SAME request via two candidates on two different, amply stocked
// listings. Capacity is never the constraint; the loser fails purely on the
// partial unique index idx_one_agreement_per_request, so it must get
// REQUEST_ALREADY_AGREED. That is the one-agreement-per-request rule, distinct
// from capacity contention.
func TestAccept_ConcurrentSameRequest_LoserGetsRequestAlreadyAgreed_FR034(t *testing.T) {
	h := newAcceptHarness(t, "accept_race_request")
	week := platform.WeekStart(acceptBaseTime)

	// Two separate listings, each with abundant capacity for the order.
	l1, sub1 := seedListing(t, h, "satu", 1000, week, week)
	l2, sub2 := seedListing(t, h, "dua", 1000, week, week)

	buyer := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyer@contoh.test", false), "Pembeli")
	req := seedRequest(t, h, buyer, 50, week)
	offer1 := seedOfferedCandidate(t, h, req, l1, subProfileID(t, h, sub1), 1_000_000)
	offer2 := seedOfferedCandidate(t, h, req, l2, subProfileID(t, h, sub2), 1_000_000)

	acc := buyerAccountOf(t, h, buyer)

	errs := runConcurrent(
		func() error { _, err := h.svc.accept(context.Background(), acc, offer1); return err },
		func() error { _, err := h.svc.accept(context.Background(), acc, offer2); return err },
	)

	assertOneWinnerOneCode(t, errs, httpx.CodeRequestAlreadyAgreed)

	// Exactly one candidate of the request reached 'agreed'.
	var agreed int
	if err := h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM request_candidate WHERE request_id = $1 AND status = 'agreed'`,
		req).Scan(&agreed); err != nil {
		t.Fatalf("hitung kandidat agreed: %v", err)
	}
	if agreed != 1 {
		t.Fatalf("kandidat agreed = %d, mau 1 (FR-034)", agreed)
	}
}

// TestAccept_UsedCapacityWithinTotalRejectsOverfill_FR079_SC018 proves the
// storage-level safety net independently of the accept path. Going through accept
// can never touch it: the sum under the lock rejects an overfill first, so the
// CHECK is never reached and such a test would prove nothing. Here RaiseUsedCapacity
// is called directly with an amount past total_capacity, and the database must
// reject it with SQLSTATE 23514.
func TestAccept_UsedCapacityWithinTotalRejectsOverfill_FR079_SC018(t *testing.T) {
	h := newAcceptHarness(t, "accept_check_overfill")
	week := platform.WeekStart(acceptBaseTime)
	listingID, _ := seedListing(t, h, "alfa", 100, week, week)

	var periodID pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`SELECT id FROM availability_period WHERE listing_id = $1 AND week_start = $2`,
		listingID, week).Scan(&periodID); err != nil {
		t.Fatalf("baca period: %v", err)
	}

	q := sqlcgen.New(h.pool)
	_, err := q.RaiseUsedCapacity(context.Background(), sqlcgen.RaiseUsedCapacityParams{
		ID:           periodID,
		UsedCapacity: 200, // total_capacity is 100
		UpdatedAt:    pgtype.Timestamptz{Time: acceptBaseTime, Valid: true},
	})
	if err == nil {
		t.Fatal("RaiseUsedCapacity melampaui total berhasil, mau ditolak constraint")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("galat = %v, mau SQLSTATE 23514 (check_violation)", err)
	}
	if pgErr.ConstraintName != "used_capacity_within_total" {
		t.Fatalf("constraint = %q, mau used_capacity_within_total", pgErr.ConstraintName)
	}
}

// TestAccept_DeadlineBeyondHorizon_Succeeds_FR088 closes the gap the optimistic
// pre-lock estimate opens: a single accept, no race, with the deadline past the
// listing's stored horizon must SUCCEED, not return CAPACITY_ALREADY_TAKEN.
// EnsureHorizon materializes the missing weeks inside the transaction, so the
// post-lock read finds real capacity that matches the optimistic estimate. This
// proves the estimate does not manufacture a false positive.
func TestAccept_DeadlineBeyondHorizon_Succeeds_FR088(t *testing.T) {
	h := newAcceptHarness(t, "accept_beyond_horizon")
	base := platform.WeekStart(acceptBaseTime)
	horizon := base            // periods exist only for the base week
	deadline := base.AddDate(0, 0, 7*4) // four weeks past the horizon

	// weekly_capacity 100; the base week alone already covers 50, and
	// EnsureHorizon adds the weeks up to the deadline at the same capacity.
	listingID, subAcc := seedListing(t, h, "alfa", 100, horizon, horizon)
	subProf := subProfileID(t, h, subAcc)

	buyer := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyer@contoh.test", false), "Pembeli")
	req := seedRequest(t, h, buyer, 50, deadline)
	offer := seedOfferedCandidate(t, h, req, listingID, subProf, 1_000_000)
	acc := buyerAccountOf(t, h, buyer)

	view, err := h.svc.accept(context.Background(), acc, offer)
	if err != nil {
		t.Fatalf("accept gagal, mau berhasil: %v", err)
	}
	if len(view.Allocations) == 0 {
		t.Fatal("tidak ada alokasi, mau minimal satu")
	}
	if got := usedCapacity(t, h, listingID); got != 50 {
		t.Fatalf("used_capacity total = %d, mau 50", got)
	}
	_ = horizon
}

// buyerAccountOf returns the user_account id that owns a buyer profile, the
// principal id the accept path checks the offer against.
func buyerAccountOf(t *testing.T, h *acceptHarness, profile pgtype.UUID) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`SELECT account_id FROM business_profile WHERE id = $1`, profile).Scan(&id); err != nil {
		t.Fatalf("lookup account: %v", err)
	}
	return id
}

// runConcurrent runs the given functions in parallel and returns their errors in
// order. A barrier releases both at once so they genuinely race on the lock.
func runConcurrent(fns ...func() error) []error {
	errs := make([]error, len(fns))
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i, fn := range fns {
		wg.Add(1)
		go func(i int, fn func() error) {
			defer wg.Done()
			<-start
			errs[i] = fn()
		}(i, fn)
	}
	close(start)
	wg.Wait()
	return errs
}

// assertOneWinnerOneCode asserts exactly one call succeeded and the other failed
// with wantCode.
func assertOneWinnerOneCode(t *testing.T, errs []error, wantCode httpx.Code) {
	t.Helper()
	var winners int
	var loserErr error
	for _, err := range errs {
		if err == nil {
			winners++
			continue
		}
		loserErr = err
	}
	if winners != 1 {
		t.Fatalf("pemenang = %d, mau tepat 1; galat = %v", winners, errs)
	}
	if got := errCode(loserErr); got != wantCode {
		t.Fatalf("kode kalah = %q, mau %q; galat = %v", got, wantCode, loserErr)
	}
}
