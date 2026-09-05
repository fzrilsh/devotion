package search

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// baseTime is a fixed Monday (2026-08-24) so WeekStart(now) is the day itself
// and every capacity-range assertion is deterministic.
var baseTime = time.Date(2026, 8, 24, 10, 3, 0, 0, time.UTC)

const dateFmt = "2006-01-02"

// phoneSeq hands each seeded account a unique numeric phone matching the
// user_account phone_format check (^62[0-9]{8,13}$); a listing name can carry
// letters, so it cannot go into the phone directly.
var phoneSeq atomic.Int64

func nextPhone() string {
	return "628" + strconv.Itoa(100000000+int(phoneSeq.Add(1)))
}

// mockAuth is a stand-in Authenticator so the search tests never import the
// account package. A nil principal is unauthenticated (401); a principal with
// the wrong Roles is a 403.
type mockAuth struct {
	principal *httpx.Principal
}

func (m *mockAuth) Authenticate(_ *http.Request) (httpx.Principal, error) {
	if m.principal == nil {
		return httpx.Principal{}, httpx.ErrUnauthenticated
	}
	return *m.principal, nil
}

// noopHorizon is a HorizonExtender that records the listings it was asked to
// extend and never writes. The read query already counts uncreated weeks
// optimistically, so a search over seeded in-range capacity passes without a
// real extension.
type noopHorizon struct {
	calls []pgtype.UUID
}

func (h *noopHorizon) EnsureHorizon(_ context.Context, _ pgx.Tx, listingID pgtype.UUID, untilWeek time.Time) (time.Time, error) {
	h.calls = append(h.calls, listingID)
	return untilWeek, nil
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// harness wires a search Service against an isolated schema and exposes the
// router handler plus the pieces a test drives.
type harness struct {
	svc       *Service
	handler   http.Handler
	pool      *pgxpool.Pool
	clock     *platform.TestClock
	auth      *mockAuth
	horizon   *noopHorizon
	buyerAcc  pgtype.UUID
	buyerProf pgtype.UUID
	productID pgtype.UUID
	machineID pgtype.UUID
}

// newHarness seeds one province and city, a buyer account with its profile (the
// searcher), and one active product plus one active machine. auth starts as
// that buyer.
func newHarness(t *testing.T, name string) *harness {
	t.Helper()
	pool := testdb.New(t, name)
	clock := platform.NewTestClock(baseTime)
	horizon := &noopHorizon{}
	svc := New(pool, clock, horizon)

	seedRegion(t, pool)
	buyerAcc := seedAccount(t, pool, "buyer@contoh.test", "628110000009", false)
	buyerProf := seedProfile(t, pool, buyerAcc, "Merek Pembeli")

	auth := &mockAuth{principal: &httpx.Principal{
		Roles:   httpx.RoleBuyer,
		Account: sqlcgen.UserAccount{ID: buyerAcc},
	}}

	r := httpx.NewRouter(quietLogger())
	svc.Register(r, auth)

	h := &harness{
		svc: svc, handler: r.Handler(), pool: pool, clock: clock,
		auth: auth, horizon: horizon, buyerAcc: buyerAcc, buyerProf: buyerProf,
	}
	h.productID = seedCatalogItem(t, pool, "product", "Kaos Oblong")
	h.machineID = seedCatalogItem(t, pool, "machine", "Mesin Jahit Lurus")
	return h
}

func seedRegion(t *testing.T, pool *pgxpool.Pool) {
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
	if _, err := pool.Exec(ctx,
		`INSERT INTO city (code, province_code, name) VALUES ('3171', '31', 'Jakarta Pusat') ON CONFLICT DO NOTHING`); err != nil {
		// province 31 may not exist; only used by the province-scope test.
		_, _ = pool.Exec(ctx, `INSERT INTO province (code, name) VALUES ('31', 'DKI Jakarta') ON CONFLICT DO NOTHING`)
		if _, err := pool.Exec(ctx,
			`INSERT INTO city (code, province_code, name) VALUES ('3171', '31', 'Jakarta Pusat') ON CONFLICT DO NOTHING`); err != nil {
			t.Fatalf("seed city 3171: %v", err)
		}
	}
}

func seedAccount(t *testing.T, pool *pgxpool.Pool, email, phone string, subcontractor bool) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO user_account (email, phone, password_hash, role_subcontractor, role_buyer, created_at, updated_at)
		 VALUES ($1, $2, 'x', $3, $4, $5, $5) RETURNING id`,
		email, phone, subcontractor, !subcontractor, baseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}
	return id
}

func seedProfile(t *testing.T, pool *pgxpool.Pool, accountID pgtype.UUID, name string) pgtype.UUID {
	return seedProfileCity(t, pool, accountID, name, "3273")
}

func seedProfileCity(t *testing.T, pool *pgxpool.Pool, accountID pgtype.UUID, name, cityCode string) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO business_profile (account_id, business_name, city_code, verified, created_at, updated_at)
		 VALUES ($1, $2, $3, false, $4, $4) RETURNING id`,
		accountID, name, cityCode, baseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	return id
}

func seedCatalogItem(t *testing.T, pool *pgxpool.Pool, kind sqlcgen.ItemType, name string) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO catalog_item (type, name, active, created_at) VALUES ($1, $2, true, $3) RETURNING id`,
		kind, name, baseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed catalog item: %v", err)
	}
	return id
}

// seedListing creates a published listing for a fresh subcontractor with the
// given weekly capacity and lead days, links the harness product and machine,
// and materializes weeks from the current week to horizonWeeks ahead each with
// full capacity. It returns the listing id.
func seedListing(t *testing.T, h *harness, name string, weeklyCapacity, leadDays, horizonWeeks int32) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	pool := h.pool

	acc := seedAccount(t, pool, "subkon-"+name+"@contoh.test", nextPhone(), true)
	prof := seedProfileCity(t, pool, acc, name, "3273")

	weekNow := platform.WeekStart(baseTime)
	horizon := weekNow.AddDate(0, 0, int(7*horizonWeeks))

	var listingID pgtype.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO capacity_listing (profile_id, weekly_capacity, readiness_lead_days, published, calendar_updated_at, horizon_until, created_at, updated_at)
		 VALUES ($1, $2, $3, true, $4, $5, $4, $4) RETURNING id`,
		prof, weeklyCapacity, leadDays, baseTime, horizon).Scan(&listingID)
	if err != nil {
		t.Fatalf("seed listing: %v", err)
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO listing_product (listing_id, item_id) VALUES ($1, $2)`,
		listingID, h.productID); err != nil {
		t.Fatalf("seed listing_product: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO listing_machine (listing_id, item_id, machine_count) VALUES ($1, $2, 1)`,
		listingID, h.machineID); err != nil {
		t.Fatalf("seed listing_machine: %v", err)
	}

	for w := weekNow; !w.After(horizon); w = w.AddDate(0, 0, 7) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO availability_period (listing_id, week_start, total_capacity, used_capacity, created_at, updated_at)
			 VALUES ($1, $2, $3, 0, $4, $4)`,
			listingID, w, weeklyCapacity, baseTime); err != nil {
			t.Fatalf("seed period %s: %v", w.Format(dateFmt), err)
		}
	}
	return listingID
}

func (h *harness) do(method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = "203.0.113.7:9999"
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

func mustStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status %d, mau %d; body %s", rec.Code, want, rec.Body.String())
	}
}

type candidateResp struct {
	ListingID                  string   `json:"listing_id"`
	ProfileID                  string   `json:"profile_id"`
	BusinessName               string   `json:"business_name"`
	Score                      int      `json:"score"`
	CityCode                   *string  `json:"city_code"`
	CityName                   *string  `json:"city_name"`
	MachineTypes               []string `json:"machine_types"`
	WeeklyCapacity             int32    `json:"weekly_capacity"`
	ReadinessWeek              string   `json:"readiness_week"`
	ReadinessLeadDays          int32    `json:"readiness_lead_days"`
	TotalCapacityUntilDeadline int64    `json:"total_capacity_until_deadline"`
	CompletedJobs              int64    `json:"completed_jobs"`
	Reputation                 struct {
		EnoughData     bool     `json:"enough_data"`
		CompletionRate *int     `json:"completion_rate"`
		AverageRating  *float64 `json:"average_rating"`
		ReviewCount    int      `json:"review_count"`
	} `json:"reputation"`
	StaleCalendar    bool `json:"stale_calendar"`
	IdentityVerified bool `json:"identity_verified"`
	Criteria         []struct {
		Name   string  `json:"name"`
		Met    bool    `json:"met"`
		Detail *string `json:"detail"`
	} `json:"criteria"`
}

type resultResp struct {
	Items      []candidateResp `json:"items"`
	Pagination struct {
		HasNext    bool    `json:"has_next"`
		NextCursor *string `json:"next_cursor"`
	} `json:"pagination"`
	RegionLevel string `json:"region_level"`
	Relaxation  *struct {
		MostRestrictive string `json:"most_restrictive"`
		Suggestion      string `json:"suggestion"`
	} `json:"relaxation"`
}

type problemResp struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Errors []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"errors"`
}

func decodeResult(t *testing.T, rec *httptest.ResponseRecorder) resultResp {
	t.Helper()
	var r resultResp
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode result %q: %v", rec.Body.String(), err)
	}
	return r
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) problemResp {
	t.Helper()
	var p problemResp
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem %q: %v", rec.Body.String(), err)
	}
	return p
}

// deadlineParam returns a YYYY-MM-DD deadline n weeks past the base week.
func deadlineParam(n int) string {
	return platform.WeekStart(baseTime).AddDate(0, 0, 7*n).Format(dateFmt)
}

// searchPath builds a /api/search query for the common city-scoped shape.
func (h *harness) searchPath(quantity, weeks int) string {
	v := url.Values{}
	v.Set("product_item_id", uuidString(h.productID))
	v.Set("machine_item_id", uuidString(h.machineID))
	v.Set("quantity", strconv.Itoa(quantity))
	v.Set("deadline", deadlineParam(weeks))
	v.Set("region_level", "city")
	v.Set("city_code", "3273")
	return "/api/search?" + v.Encode()
}

// seedAgreedOrder builds the request -> candidate -> offer -> work_order chain
// for one subcontractor profile with the given work_order status, so a test can
// drive the completion-rate divisor and numerator (FR-071/FR-072). It mirrors
// the seedReview chain in determinism_test.go but leaves the status to the
// caller. It returns the work_order id.
func seedAgreedOrder(t *testing.T, h *harness, listingID, subconProfile pgtype.UUID, status string) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	weekNow := platform.WeekStart(baseTime)
	deadline := weekNow.AddDate(0, 0, 7*4)

	var requestID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO quota_request (buyer_id, product_item_id, quantity, material, deadline, reply_due_at, created_at)
		 VALUES ($1, $2, 100, 'katun', $3, $4, $5) RETURNING id`,
		h.buyerProf, h.productID, deadline, baseTime.Add(48*time.Hour), baseTime).Scan(&requestID); err != nil {
		t.Fatalf("seed quota_request: %v", err)
	}

	var candidateID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO request_candidate (request_id, listing_id, subcontractor_id, status, updated_at)
		 VALUES ($1, $2, $3, 'agreed', $4) RETURNING id`,
		requestID, listingID, subconProfile, baseTime).Scan(&candidateID); err != nil {
		t.Fatalf("seed request_candidate: %v", err)
	}

	var offerID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO offer (candidate_id, sequence, proposed_by, total_price, readiness_lead_days, created_at)
		 VALUES ($1, 1, 'subcontractor', 100000, 7, $2) RETURNING id`,
		candidateID, baseTime).Scan(&offerID); err != nil {
		t.Fatalf("seed offer: %v", err)
	}

	var orderID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO work_order (candidate_id, offer_id, buyer_id, subcontractor_id, quantity, total_price,
		     deadline, readiness_week_start, status, created_at)
		 VALUES ($1, $2, $3, $4, 100, 100000, $5, $6, $7, $8) RETURNING id`,
		candidateID, offerID, h.buyerProf, subconProfile, deadline, weekNow, status, baseTime).Scan(&orderID); err != nil {
		t.Fatalf("seed work_order: %v", err)
	}
	return orderID
}

// TestSearch_CandidateCarriesFR027Attributes_FR027_FR026 seeds one matching
// listing and asserts every informative attribute FR-027 requires is present in
// the serialized candidate: city, machine types, weekly and total capacity,
// readiness, and the reputation block. FR-026 requires the four criteria to be
// reported per candidate, so the criteria array is checked too.
func TestSearch_CandidateCarriesFR027Attributes_FR027_FR026(t *testing.T) {
	h := newHarness(t, "search_fr027")
	listingID := seedListing(t, h, "konveksi-bandung", 120, 14, 8)

	// Manual step 2.6 compares a verified against an unverified candidate, so the
	// badge must round-trip. Mark this listing's owner verified.
	prof := profileOfListing(t, h, listingID)
	if _, err := h.pool.Exec(context.Background(),
		`UPDATE business_profile SET verified = true WHERE id = $1`, prof); err != nil {
		t.Fatalf("set verified: %v", err)
	}

	res := decodeResult(t, h.do(http.MethodGet, h.searchPath(50, 8)))
	if len(res.Items) != 1 {
		t.Fatalf("mau 1 kandidat, dapat %d", len(res.Items))
	}
	got := itemByListing(t, res, listingID)

	if !got.IdentityVerified {
		t.Fatalf("identity_verified mau true untuk profil terverifikasi")
	}
	if got.CityCode == nil || *got.CityCode != "3273" {
		t.Fatalf("city_code %v, mau 3273", got.CityCode)
	}
	if got.CityName == nil || *got.CityName != "Kota Bandung" {
		t.Fatalf("city_name %v, mau Kota Bandung", got.CityName)
	}
	if len(got.MachineTypes) != 1 || got.MachineTypes[0] != "Mesin Jahit Lurus" {
		t.Fatalf("machine_types %v, mau [Mesin Jahit Lurus]", got.MachineTypes)
	}
	if got.WeeklyCapacity != 120 {
		t.Fatalf("weekly_capacity %d, mau 120", got.WeeklyCapacity)
	}
	if got.ReadinessLeadDays != 14 {
		t.Fatalf("readiness_lead_days %d, mau 14", got.ReadinessLeadDays)
	}
	// 14-day lead from the Monday base lands readiness on the Monday two weeks
	// out, so readiness_week is that date in YYYY-MM-DD form.
	wantWeek := platform.WeekStart(baseTime).AddDate(0, 0, 14).Format(dateFmt)
	if got.ReadinessWeek != wantWeek {
		t.Fatalf("readiness_week %q, mau %q", got.ReadinessWeek, wantWeek)
	}
	// weeks 2..8 inclusive at 120 = 7*120 = 840.
	if got.TotalCapacityUntilDeadline != 7*120 {
		t.Fatalf("total_capacity_until_deadline %d, mau 840", got.TotalCapacityUntilDeadline)
	}
	if len(got.Criteria) != 4 {
		t.Fatalf("mau 4 criteria, dapat %d", len(got.Criteria))
	}
	// A brand-new subcontractor has no orders, so reputation is below the FR-073
	// threshold: no percentage, no rating, no reviews.
	if got.Reputation.EnoughData {
		t.Fatalf("enough_data mau false untuk subkon baru")
	}
	if got.Reputation.CompletionRate != nil {
		t.Fatalf("completion_rate mau nil di bawah ambang, dapat %d", *got.Reputation.CompletionRate)
	}
	if got.Reputation.ReviewCount != 0 {
		t.Fatalf("review_count mau 0, dapat %d", got.Reputation.ReviewCount)
	}
	if got.CompletedJobs != 0 {
		t.Fatalf("completed_jobs mau 0, dapat %d", got.CompletedJobs)
	}
}

// TestSearch_CompletionRateRespectsEnoughDataThreshold_FR073 proves the search
// result honors the FR-073 cutoff: a subcontractor below three agreed orders
// reports enough_data false with a nil completion_rate, while one at or above
// three reports enough_data true and a concrete percentage. Reputation is read
// at query time, never materialized, so seeding orders alone changes the result.
func TestSearch_CompletionRateRespectsEnoughDataThreshold_FR073(t *testing.T) {
	h := newHarness(t, "search_fr073")
	below := seedListing(t, h, "belum-cukup", 100, 7, 8)
	above := seedListing(t, h, "cukup-data", 100, 7, 8)

	belowProf := profileOfListing(t, h, below)
	aboveProf := profileOfListing(t, h, above)

	// Below the threshold: two confirmed orders, divisor 2 < 3.
	seedAgreedOrder(t, h, below, belowProf, "confirmed")
	seedAgreedOrder(t, h, below, belowProf, "confirmed")

	// At the threshold: three orders, all confirmed, divisor 3 -> 100 percent.
	seedAgreedOrder(t, h, above, aboveProf, "confirmed")
	seedAgreedOrder(t, h, above, aboveProf, "confirmed")
	seedAgreedOrder(t, h, above, aboveProf, "confirmed")

	res := decodeResult(t, h.do(http.MethodGet, h.searchPath(50, 4)))

	belowCand := itemByListing(t, res, below)
	if belowCand.Reputation.EnoughData {
		t.Fatalf("enough_data mau false untuk 2 pesanan")
	}
	if belowCand.Reputation.CompletionRate != nil {
		t.Fatalf("completion_rate mau nil di bawah ambang, dapat %d", *belowCand.Reputation.CompletionRate)
	}

	aboveCand := itemByListing(t, res, above)
	if !aboveCand.Reputation.EnoughData {
		t.Fatalf("enough_data mau true untuk 3 pesanan")
	}
	if aboveCand.Reputation.CompletionRate == nil {
		t.Fatalf("completion_rate mau terisi di atas ambang, dapat nil")
	}
	if *aboveCand.Reputation.CompletionRate != 100 {
		t.Fatalf("completion_rate %d, mau 100", *aboveCand.Reputation.CompletionRate)
	}
	// completed_jobs counts confirmed orders where the profile is subcontractor.
	if aboveCand.CompletedJobs != 3 {
		t.Fatalf("completed_jobs %d, mau 3", aboveCand.CompletedJobs)
	}
}

// TestSearch_HappyPath_ReturnsScoredCandidate_FR022_FR023 seeds one matching
// listing and asserts the buyer gets it back with a perfect score of 4.
func TestSearch_HappyPath_ReturnsScoredCandidate_FR022_FR023(t *testing.T) {
	h := newHarness(t, "search_happy")
	listingID := seedListing(t, h, "alfa", 100, 7, 8)

	rec := h.do(http.MethodGet, h.searchPath(50, 4))
	mustStatus(t, rec, http.StatusOK)

	res := decodeResult(t, rec)
	if len(res.Items) != 1 {
		t.Fatalf("mau 1 item, dapat %d; body %s", len(res.Items), rec.Body.String())
	}
	got := res.Items[0]
	if got.ListingID != uuidString(listingID) {
		t.Fatalf("listing_id %q, mau %q", got.ListingID, uuidString(listingID))
	}
	if got.Score != 4 {
		t.Fatalf("score %d, mau 4", got.Score)
	}
	if len(got.Criteria) != 4 {
		t.Fatalf("mau 4 criteria, dapat %d", len(got.Criteria))
	}
	for _, c := range got.Criteria {
		if !c.Met {
			t.Fatalf("kriteria %q tidak terpenuhi", c.Name)
		}
	}
	if res.RegionLevel != "city" {
		t.Fatalf("region_level %q, mau city", res.RegionLevel)
	}
}

// TestSearch_ExcludesOwnListing_FR081 confirms a listing owned by the searcher's
// own profile never appears in the buyer's own results.
func TestSearch_ExcludesOwnListing_FR081(t *testing.T) {
	h := newHarness(t, "search_own")
	// A listing whose profile is the buyer's own profile.
	weekNow := platform.WeekStart(baseTime)
	horizon := weekNow.AddDate(0, 0, 7*8)
	var listingID pgtype.UUID
	err := h.pool.QueryRow(context.Background(),
		`INSERT INTO capacity_listing (profile_id, weekly_capacity, readiness_lead_days, published, calendar_updated_at, horizon_until, created_at, updated_at)
		 VALUES ($1, 100, 7, true, $2, $3, $2, $2) RETURNING id`,
		h.buyerProf, baseTime, horizon).Scan(&listingID)
	if err != nil {
		t.Fatalf("seed own listing: %v", err)
	}

	rec := h.do(http.MethodGet, h.searchPath(50, 4))
	mustStatus(t, rec, http.StatusOK)
	res := decodeResult(t, rec)
	if len(res.Items) != 0 {
		t.Fatalf("mau 0 item (listing sendiri dikecualikan), dapat %d", len(res.Items))
	}
}

// TestSearch_RejectsNonBuyer_FR022 proves the endpoint gate rejects a caller
// without the buyer role with 403.
func TestSearch_RejectsNonBuyer_FR022(t *testing.T) {
	h := newHarness(t, "search_role")
	h.auth.principal = &httpx.Principal{
		Roles:   httpx.RoleSubcontractor,
		Account: sqlcgen.UserAccount{ID: h.buyerAcc},
	}
	rec := h.do(http.MethodGet, h.searchPath(50, 4))
	mustStatus(t, rec, http.StatusForbidden)
}

// TestSearch_RejectsUnauthenticated_FR022 proves a missing session is 401.
func TestSearch_RejectsUnauthenticated_FR022(t *testing.T) {
	h := newHarness(t, "search_noauth")
	h.auth.principal = nil
	rec := h.do(http.MethodGet, h.searchPath(50, 4))
	mustStatus(t, rec, http.StatusUnauthorized)
}

// TestSearch_RejectsInvalidInput_FR022 covers the required-param and format
// validation paths, each returning 422.
func TestSearch_RejectsInvalidInput_FR022(t *testing.T) {
	h := newHarness(t, "search_invalid")

	cases := map[string]string{
		"missing product": "/api/search?quantity=1&deadline=" + deadlineParam(4) + "&region_level=city&city_code=3273",
		"missing deadline": "/api/search?product_item_id=" + uuidString(h.productID) +
			"&quantity=1&region_level=city&city_code=3273",
		"bad quantity": "/api/search?product_item_id=" + uuidString(h.productID) +
			"&quantity=0&deadline=" + deadlineParam(4) + "&region_level=city&city_code=3273",
		"bad deadline": "/api/search?product_item_id=" + uuidString(h.productID) +
			"&quantity=1&deadline=bukan-tanggal&region_level=city&city_code=3273",
		"past deadline": "/api/search?product_item_id=" + uuidString(h.productID) +
			"&quantity=1&deadline=2026-08-02&region_level=city&city_code=3273",
		"missing city_code": "/api/search?product_item_id=" + uuidString(h.productID) +
			"&quantity=1&deadline=" + deadlineParam(4) + "&region_level=city",
		"bad region_level": "/api/search?product_item_id=" + uuidString(h.productID) +
			"&quantity=1&deadline=" + deadlineParam(4) + "&region_level=galaxy",
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			rec := h.do(http.MethodGet, path)
			mustStatus(t, rec, http.StatusUnprocessableEntity)
		})
	}
}

// TestSearch_EmptyAtNational_SuggestsRelaxation_FR028 asserts that when nothing
// matches at the national level, the body names the most restrictive filter and
// a concrete relaxation.
func TestSearch_EmptyAtNational_SuggestsRelaxation_FR028(t *testing.T) {
	h := newHarness(t, "search_relax")
	// No listing is seeded, so the national scope, the highest level, returns
	// nothing. The query ranks every in-region published listing (partial
	// matches included, FR-023/FR-026), so an empty result means the scope is
	// genuinely empty, which is when FR-028 names the most restrictive filter.

	v := url.Values{}
	v.Set("product_item_id", uuidString(h.productID))
	v.Set("quantity", "100000")
	v.Set("deadline", deadlineParam(4))
	v.Set("region_level", "national")
	rec := h.do(http.MethodGet, "/api/search?"+v.Encode())
	mustStatus(t, rec, http.StatusOK)

	res := decodeResult(t, rec)
	if len(res.Items) != 0 {
		t.Fatalf("mau 0 item, dapat %d", len(res.Items))
	}
	if res.Relaxation == nil {
		t.Fatalf("mau saran pelonggaran, dapat nil; body %s", rec.Body.String())
	}
	if res.Relaxation.MostRestrictive == "" || res.Relaxation.Suggestion == "" {
		t.Fatalf("saran pelonggaran tidak lengkap: %+v", res.Relaxation)
	}
}

// TestSearch_KeysetPagination_StableAcrossPages_FR080 seeds more than one page
// of candidates and walks the opaque cursor, asserting no overlap and no gap.
func TestSearch_KeysetPagination_StableAcrossPages_FR080(t *testing.T) {
	h := newHarness(t, "search_paging")
	const total = 5
	for i := 0; i < total; i++ {
		// Vary weekly capacity so remaining_capacity differs and ordering is
		// exercised beyond score alone.
		seedListing(t, h, "subkon-"+string(rune('a'+i)), int32(100+i), 7, 8)
	}

	seen := map[string]bool{}
	path := h.searchPath(10, 4) + "&size=2"
	for {
		rec := h.do(http.MethodGet, path)
		mustStatus(t, rec, http.StatusOK)
		res := decodeResult(t, rec)
		for _, it := range res.Items {
			if seen[it.ListingID] {
				t.Fatalf("listing %s muncul dua kali antar halaman", it.ListingID)
			}
			seen[it.ListingID] = true
		}
		if !res.Pagination.HasNext {
			break
		}
		if res.Pagination.NextCursor == nil {
			t.Fatalf("has_next true tapi next_cursor nil")
		}
		v := url.Values{}
		v.Set("product_item_id", uuidString(h.productID))
		v.Set("machine_item_id", uuidString(h.machineID))
		v.Set("quantity", "10")
		v.Set("deadline", deadlineParam(4))
		v.Set("region_level", "city")
		v.Set("city_code", "3273")
		v.Set("size", "2")
		v.Set("cursor", *res.Pagination.NextCursor)
		path = "/api/search?" + v.Encode()
	}
	if len(seen) != total {
		t.Fatalf("mau %d listing unik, dapat %d", total, len(seen))
	}
}

// TestSearchRoutes_TidakMeninggalkanRuteTanpaKeputusanPeran proves the search
// route sits behind a role gate, so the coverage gate cannot pass while the
// endpoint ships without a role decision. It runs without a database: Register
// only records patterns.
func TestSearchRoutes_TidakMeninggalkanRuteTanpaKeputusanPeran(t *testing.T) {
	svc := &Service{}
	r := httpx.NewRouter(quietLogger())
	svc.Register(r, &mockAuth{})

	if got := r.UncoveredAPIRoutes(); len(got) != 0 {
		t.Fatalf("rute search tak tercakup = %v, mau kosong", got)
	}
}
