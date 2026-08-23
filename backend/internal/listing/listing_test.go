package listing

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// baseTime is a fixed Monday (2026-08-24) so WeekStart(now) is the day itself
// and every calendar assertion is deterministic. It matches the account
// package harness so both suites reason about the same week.
var baseTime = time.Date(2026, 8, 24, 10, 3, 0, 0, time.UTC)

// dateFmt is the wire date layout the period and listing responses use.
const dateFmt = "2006-01-02"

// mockAuth is a stand-in Authenticator so the listing tests never import the
// account package. principal is what Authenticate returns; a nil principal is
// an unauthenticated request (401), a principal with the wrong Roles is a 403.
type mockAuth struct {
	principal *httpx.Principal
}

func (m *mockAuth) Authenticate(_ *http.Request) (httpx.Principal, error) {
	if m.principal == nil {
		return httpx.Principal{}, httpx.ErrUnauthenticated
	}
	return *m.principal, nil
}

// quietLogger silences the router log so test output stays readable.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// harness wires a Service against an isolated schema and exposes the router
// handler plus the pieces a test drives.
type harness struct {
	svc       *Service
	handler   http.Handler
	pool      *pgxpool.Pool
	clock     *platform.TestClock
	auth      *mockAuth
	accountID pgtype.UUID
	profileID pgtype.UUID
	productID string
	machineID string
}

// newHarness wires a Service against an isolated schema, seeds one province and
// city, a subcontractor account with its profile, and one active product plus
// one active machine every listing test reuses. auth starts as that
// subcontractor; a test that checks a role gate swaps auth.principal.
func newHarness(t *testing.T, name string) *harness {
	t.Helper()
	pool := testdb.New(t, name)
	clock := platform.NewTestClock(baseTime)
	svc := New(pool, clock)

	seedRegion(t, pool)
	accountID := seedAccount(t, pool, "subkon@contoh.test", "628110000001", true)
	profileID := seedProfile(t, pool, accountID, "Konveksi Subkon")

	auth := &mockAuth{principal: &httpx.Principal{
		Roles:   httpx.RoleSubcontractor,
		Account: sqlcgen.UserAccount{ID: accountID},
	}}

	r := httpx.NewRouter(quietLogger())
	svc.Register(r, auth)

	h := &harness{svc: svc, handler: r.Handler(), pool: pool, clock: clock, auth: auth}
	h.accountID = accountID
	h.profileID = profileID
	h.productID = seedCatalogItem(t, pool, sqlcgen.ItemTypeProduct, "Kaos Oblong")
	h.machineID = seedCatalogItem(t, pool, sqlcgen.ItemTypeMachine, "Mesin Jahit Lurus")
	return h
}

// seedRegion inserts the one province and city a profile points at. Idempotent
// so a reused schema does not error on a second run.
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
}

// seedAccount inserts a user_account with the subcontractor or buyer role and
// returns its id. created_at/updated_at have no default, so they are supplied.
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

// seedProfile inserts the business_profile born with an account and returns its id.
func seedProfile(t *testing.T, pool *pgxpool.Pool, accountID pgtype.UUID, name string) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO business_profile (account_id, business_name, city_code, created_at, updated_at)
		 VALUES ($1, $2, '3273', $3, $3) RETURNING id`,
		accountID, name, baseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	return id
}

// seedCatalogItem inserts an active catalog item of the given type and returns
// its id as canonical text, which is what listing requests carry.
func seedCatalogItem(t *testing.T, pool *pgxpool.Pool, kind sqlcgen.ItemType, name string) string {
	t.Helper()
	var id pgtype.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO catalog_item (type, name, active, created_at) VALUES ($1, $2, true, $3) RETURNING id`,
		kind, name, baseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed catalog item: %v", err)
	}
	return uuidString(id)
}

// periodID returns the availability_period id for a listing week, failing the
// test if the week has no row yet.
func (h *harness) periodID(t *testing.T, listingID pgtype.UUID, week time.Time) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	err := h.pool.QueryRow(context.Background(),
		`SELECT id FROM availability_period WHERE listing_id = $1 AND week_start = $2`,
		listingID, week).Scan(&id)
	if err != nil {
		t.Fatalf("lookup period %s: %v", week.Format(dateFmt), err)
	}
	return id
}

// seedAllocation puts a real, unreversed capacity_allocation on a period so the
// FR-089 propagation and mark-full guards see a live order. It builds the whole
// chain (buyer, quota_request, request_candidate, offer, work_order) with direct
// SQL rather than through internal/order, keeping the dependency order -> listing.
// The work order's readiness week is set to week so the FR-087 trigger admits
// the allocation. used_capacity is bumped to match so the capacity guards fire.
func (h *harness) seedAllocation(t *testing.T, listingID, subconProfileID pgtype.UUID, week time.Time, quantity int32) {
	t.Helper()
	ctx := context.Background()
	pool := h.pool

	buyerAccount := seedAccount(t, pool, "buyer-"+week.Format(dateFmt)+"@contoh.test", "628990000001", false)
	buyerProfile := seedProfile(t, pool, buyerAccount, "Merek Pembeli")

	var productID pgtype.UUID
	if err := pool.QueryRow(ctx,
		`SELECT id FROM catalog_item WHERE type = 'product' LIMIT 1`).Scan(&productID); err != nil {
		t.Fatalf("lookup product: %v", err)
	}

	var requestID pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO quota_request (buyer_id, product_item_id, quantity, material, deadline, reply_due_at, created_at)
		 VALUES ($1, $2, $3, 'katun', $4, $5, $6) RETURNING id`,
		buyerProfile, productID, quantity, week.AddDate(0, 0, 7), baseTime.AddDate(0, 0, 1), baseTime).Scan(&requestID); err != nil {
		t.Fatalf("seed quota_request: %v", err)
	}

	var candidateID pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO request_candidate (request_id, listing_id, subcontractor_id, status, updated_at)
		 VALUES ($1, $2, $3, 'agreed', $4) RETURNING id`,
		requestID, listingID, subconProfileID, baseTime).Scan(&candidateID); err != nil {
		t.Fatalf("seed request_candidate: %v", err)
	}

	var offerID pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO offer (candidate_id, sequence, proposed_by, total_price, readiness_lead_days, created_at)
		 VALUES ($1, 1, 'subcontractor', 1000000, 7, $2) RETURNING id`,
		candidateID, baseTime).Scan(&offerID); err != nil {
		t.Fatalf("seed offer: %v", err)
	}

	var workOrderID pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO work_order (candidate_id, offer_id, buyer_id, subcontractor_id, quantity, total_price, deadline, readiness_week_start, created_at)
		 VALUES ($1, $2, $3, $4, $5, 1000000, $6, $7, $8) RETURNING id`,
		candidateID, offerID, buyerProfile, subconProfileID, quantity, week.AddDate(0, 0, 7), week, baseTime).Scan(&workOrderID); err != nil {
		t.Fatalf("seed work_order: %v", err)
	}

	periodID := h.periodID(t, listingID, week)
	if _, err := pool.Exec(ctx,
		`INSERT INTO capacity_allocation (work_order_id, period_id, quantity, created_at)
		 VALUES ($1, $2, $3, $4)`,
		workOrderID, periodID, quantity, baseTime); err != nil {
		t.Fatalf("seed capacity_allocation: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE availability_period SET used_capacity = $2 WHERE id = $1`,
		periodID, quantity); err != nil {
		t.Fatalf("bump used_capacity: %v", err)
	}
}

// do sends a JSON request through the router and returns the recorder. The mock
// authenticator supplies the principal, so no cookie is needed.
func (h *harness) do(method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.RemoteAddr = "203.0.113.7:9999"
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

// decode unmarshals a JSON response body into dst, failing the test on error.
func decode(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
}

// listingResp mirrors listingView for decoding a listing response.
type listingResp struct {
	ListingID         string     `json:"listing_id"`
	ProfileID         string     `json:"profile_id"`
	WeeklyCapacity    int32      `json:"weekly_capacity"`
	ReadinessLeadDays int32      `json:"readiness_lead_days"`
	Published         bool       `json:"published"`
	CalendarUpdatedAt *time.Time `json:"calendar_updated_at"`
	HorizonUntil      string     `json:"horizon_until"`
	ProductItems      []struct {
		ItemID string `json:"item_id"`
		Kind   string `json:"kind"`
		Name   string `json:"name"`
		Active bool   `json:"active"`
	} `json:"product_items"`
	Machines []struct {
		Item struct {
			ItemID string `json:"item_id"`
		} `json:"item"`
		MachineCount int32 `json:"machine_count"`
	} `json:"machines"`
}

// periodResp mirrors periodView for decoding a periods response.
type periodResp struct {
	WeekStart  string `json:"week_start"`
	Capacity   int32  `json:"capacity"`
	Allocated  int64  `json:"allocated"`
	Remaining  int64  `json:"remaining"`
	MarkedFull bool   `json:"marked_full"`
}

// validCreate returns a well-formed ListingRequest body using the seeded items.
func (h *harness) validCreate(capacity int32) map[string]any {
	return map[string]any{
		"weekly_capacity":     capacity,
		"readiness_lead_days": 7,
		"product_item_ids":    []string{h.productID},
		"machines":            []map[string]any{{"item_id": h.machineID, "machine_count": 2}},
	}
}

// listingRow reads the listing id of the harness profile straight from the DB.
func (h *harness) listingRow(t *testing.T) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	err := h.pool.QueryRow(context.Background(),
		`SELECT id FROM capacity_listing WHERE profile_id = $1`, h.profileID).Scan(&id)
	if err != nil {
		t.Fatalf("lookup listing: %v", err)
	}
	return id
}

// mondayAfter returns the Monday n weeks past the base week.
func mondayAfter(n int) time.Time {
	return platform.WeekStart(baseTime).AddDate(0, 0, 7*n)
}

// problemResp captures the fields of an application/problem+json body the tests
// assert on: the machine code and, for validation failures, the per-field list.
type problemResp struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Errors []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"errors"`
}

// problemBody decodes a problem+json response so a test can assert its code.
func problemBody(t *testing.T, rec *httptest.ResponseRecorder) problemResp {
	t.Helper()
	var p problemResp
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem %q: %v", rec.Body.String(), err)
	}
	return p
}

// mustStatus fails the test unless the recorder carries want, quoting the body.
func mustStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status %d, mau %d; body %s", rec.Code, want, rec.Body.String())
	}
}

// TestCreateListing_JalurBerhasil_FR012 is the happy path: a subcontractor
// creates their single listing and gets it back with its items echoed.
func TestCreateListing_JalurBerhasil_FR012(t *testing.T) {
	h := newHarness(t, "create_ok")
	rec := h.do("POST", "/api/listing/me", h.validCreate(500))
	mustStatus(t, rec, http.StatusCreated)

	var got listingResp
	decode(t, rec, &got)
	if got.WeeklyCapacity != 500 {
		t.Errorf("weekly_capacity = %d, mau 500", got.WeeklyCapacity)
	}
	if got.ReadinessLeadDays != 7 {
		t.Errorf("readiness_lead_days = %d, mau 7", got.ReadinessLeadDays)
	}
	if len(got.ProductItems) != 1 || got.ProductItems[0].ItemID != h.productID {
		t.Errorf("product_items = %+v, mau satu %s", got.ProductItems, h.productID)
	}
	if len(got.Machines) != 1 || got.Machines[0].Item.ItemID != h.machineID || got.Machines[0].MachineCount != 2 {
		t.Errorf("machines = %+v, mau satu %s x2", got.Machines, h.machineID)
	}
	if got.ProfileID != uuidString(h.profileID) {
		t.Errorf("profile_id = %s, mau %s", got.ProfileID, uuidString(h.profileID))
	}
}

// TestCreateListing_AtributWajibKosong_MenyebutKolomnya_FR013 proves a body
// missing products and machines is a 422 naming both fields.
func TestCreateListing_AtributWajibKosong_MenyebutKolomnya_FR013(t *testing.T) {
	h := newHarness(t, "create_wajib")
	rec := h.do("POST", "/api/listing/me", map[string]any{
		"weekly_capacity":     500,
		"readiness_lead_days": 7,
		"product_item_ids":    []string{},
		"machines":            []map[string]any{},
	})
	mustStatus(t, rec, http.StatusUnprocessableEntity)
	p := problemBody(t, rec)
	if p.Code != "VALIDATION_FAILED" {
		t.Fatalf("code = %s, mau VALIDATION_FAILED", p.Code)
	}
	seen := map[string]bool{}
	for _, e := range p.Errors {
		seen[e.Field] = true
	}
	if !seen["product_item_ids"] || !seen["machines"] {
		t.Errorf("errors = %+v, mau menyebut product_item_ids dan machines", p.Errors)
	}
}

// TestCreateListing_KapasitasNol_Ditolak_FR014 proves capacity below 1 is a 422
// naming weekly_capacity.
func TestCreateListing_KapasitasNol_Ditolak_FR014(t *testing.T) {
	h := newHarness(t, "create_nol")
	rec := h.do("POST", "/api/listing/me", h.validCreate(0))
	mustStatus(t, rec, http.StatusUnprocessableEntity)
	p := problemBody(t, rec)
	found := false
	for _, e := range p.Errors {
		if e.Field == "weekly_capacity" {
			found = true
		}
	}
	if !found {
		t.Errorf("errors = %+v, mau menyebut weekly_capacity", p.Errors)
	}
}

// TestCreateListing_ItemMesinSebagaiProduk_Ditolak_FR012 proves a machine id
// sent as a product is a 422 naming product_item_ids, not the trigger's bare 500.
func TestCreateListing_ItemMesinSebagaiProduk_Ditolak_FR012(t *testing.T) {
	h := newHarness(t, "create_salahtipe")
	rec := h.do("POST", "/api/listing/me", map[string]any{
		"weekly_capacity":     500,
		"readiness_lead_days": 7,
		"product_item_ids":    []string{h.machineID},
		"machines":            []map[string]any{{"item_id": h.machineID, "machine_count": 2}},
	})
	mustStatus(t, rec, http.StatusUnprocessableEntity)
	p := problemBody(t, rec)
	found := false
	for _, e := range p.Errors {
		if e.Field == "product_item_ids" {
			found = true
		}
	}
	if !found {
		t.Errorf("errors = %+v, mau menyebut product_item_ids", p.Errors)
	}
}

// TestCreateListing_TanpaPengajuanVerifikasi_TetapTayang_FR010 proves a fresh
// listing is published immediately, with no verification gate. The source
// document places a "Menunggu Verifikasi" status on the listing flow; our spec
// deliberately deviates (FR-010: a listing is searchable regardless of identity
// verification), and this test locks that decision in.
func TestCreateListing_TanpaPengajuanVerifikasi_TetapTayang_FR010(t *testing.T) {
	h := newHarness(t, "create_tayang")
	rec := h.do("POST", "/api/listing/me", h.validCreate(500))
	mustStatus(t, rec, http.StatusCreated)
	var got listingResp
	decode(t, rec, &got)
	if !got.Published {
		t.Error("published = false, mau true (FR-010 tayang tanpa verifikasi)")
	}
}

// TestCreateListing_ProfilSudahPunyaListing_Conflict_FR012 proves a second
// create from the same profile is a 409 LISTING_ALREADY_EXISTS.
func TestCreateListing_ProfilSudahPunyaListing_Conflict_FR012(t *testing.T) {
	h := newHarness(t, "create_dobel")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)
	rec := h.do("POST", "/api/listing/me", h.validCreate(400))
	mustStatus(t, rec, http.StatusConflict)
	if p := problemBody(t, rec); p.Code != "LISTING_ALREADY_EXISTS" {
		t.Errorf("code = %s, mau LISTING_ALREADY_EXISTS", p.Code)
	}
}

// TestCreateListing_TanpaPeranSubkontraktor_Forbidden_FR012 proves a caller
// without the subcontractor role is rejected before the handler runs.
func TestCreateListing_TanpaPeranSubkontraktor_Forbidden_FR012(t *testing.T) {
	h := newHarness(t, "create_peran")
	h.auth.principal = &httpx.Principal{
		Roles:   httpx.RoleBuyer,
		Account: sqlcgen.UserAccount{ID: h.accountID},
	}
	rec := h.do("POST", "/api/listing/me", h.validCreate(500))
	mustStatus(t, rec, http.StatusForbidden)
}

// TestGetListing_BelumAda_NotFound_FR012 proves GET before any create is a 404
// the owner resolves by creating one.
func TestGetListing_BelumAda_NotFound_FR012(t *testing.T) {
	h := newHarness(t, "get_kosong")
	rec := h.do("GET", "/api/listing/me", nil)
	mustStatus(t, rec, http.StatusNotFound)
	if p := problemBody(t, rec); p.Code != "LISTING_NOT_FOUND" {
		t.Errorf("code = %s, mau LISTING_NOT_FOUND", p.Code)
	}
}

// TestUpdateListing_KapasitasBaruDiteruskanKePeriodeMendatang_FR089 proves a new
// weekly capacity propagates to future unallocated periods.
func TestUpdateListing_KapasitasBaruDiteruskanKePeriodeMendatang_FR089(t *testing.T) {
	h := newHarness(t, "update_propagasi")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	body := h.validCreate(300)
	rec := h.do("PUT", "/api/listing/me", body)
	mustStatus(t, rec, http.StatusOK)

	// A future week must now carry the new 300 capacity.
	listingID := h.listingRow(t)
	week := mondayAfter(3)
	var cap int32
	if err := h.pool.QueryRow(context.Background(),
		`SELECT total_capacity FROM availability_period WHERE listing_id = $1 AND week_start = $2`,
		listingID, week).Scan(&cap); err != nil {
		t.Fatalf("lookup period: %v", err)
	}
	if cap != 300 {
		t.Errorf("total_capacity minggu depan = %d, mau 300", cap)
	}
}

// TestUpdateListing_PeriodeTeralokasiTetapSepertiSemula_FR089 proves a period
// with a live allocation keeps its agreed capacity when the listing capacity
// changes around it.
func TestUpdateListing_PeriodeTeralokasiTetapSepertiSemula_FR089(t *testing.T) {
	h := newHarness(t, "update_teralokasi")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	listingID := h.listingRow(t)
	week := mondayAfter(2)
	h.seedAllocation(t, listingID, h.profileID, week, 200)

	// Raise the base capacity; the allocated week must not be rewritten to 600.
	mustStatus(t, h.do("PUT", "/api/listing/me", h.validCreate(600)), http.StatusOK)

	var cap int32
	if err := h.pool.QueryRow(context.Background(),
		`SELECT total_capacity FROM availability_period WHERE listing_id = $1 AND week_start = $2`,
		listingID, week).Scan(&cap); err != nil {
		t.Fatalf("lookup period: %v", err)
	}
	if cap != 500 {
		t.Errorf("periode teralokasi total_capacity = %d, mau tetap 500", cap)
	}
}

// TestUpdateListing_KapasitasDiBawahYangTerpakai_Conflict_FR089 proves lowering
// the listing capacity below a future period's used capacity is a 409.
func TestUpdateListing_KapasitasDiBawahYangTerpakai_Conflict_FR089(t *testing.T) {
	h := newHarness(t, "update_konflik")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	listingID := h.listingRow(t)
	week := mondayAfter(2)
	h.seedAllocation(t, listingID, h.profileID, week, 400)

	rec := h.do("PUT", "/api/listing/me", h.validCreate(300))
	mustStatus(t, rec, http.StatusConflict)
	if p := problemBody(t, rec); p.Code != "CAPACITY_ALREADY_ALLOCATED" {
		t.Errorf("code = %s, mau CAPACITY_ALREADY_ALLOCATED", p.Code)
	}
}

// TestUpdateListing_PeriodeLampauTidakTersentuh_FR089 proves a past week keeps
// its capacity when the listing capacity is propagated forward.
func TestUpdateListing_PeriodeLampauTidakTersentuh_FR089(t *testing.T) {
	h := newHarness(t, "update_lampau")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	listingID := h.listingRow(t)
	// Insert a past week directly; propagation must not rewrite it.
	pastWeek := platform.WeekStart(baseTime).AddDate(0, 0, -7)
	if _, err := h.pool.Exec(context.Background(),
		`INSERT INTO availability_period (listing_id, week_start, total_capacity, created_at, updated_at)
		 VALUES ($1, $2, 500, $3, $3)`, listingID, pastWeek, baseTime); err != nil {
		t.Fatalf("seed past period: %v", err)
	}

	mustStatus(t, h.do("PUT", "/api/listing/me", h.validCreate(300)), http.StatusOK)

	var cap int32
	if err := h.pool.QueryRow(context.Background(),
		`SELECT total_capacity FROM availability_period WHERE listing_id = $1 AND week_start = $2`,
		listingID, pastWeek).Scan(&cap); err != nil {
		t.Fatalf("lookup past period: %v", err)
	}
	if cap != 500 {
		t.Errorf("periode lampau total_capacity = %d, mau tetap 500", cap)
	}
}

// TestSetVisibility_NonaktifDanAktifkanKembali_FR015 proves a listing can be
// hidden then re-published, both 200.
func TestSetVisibility_NonaktifDanAktifkanKembali_FR015(t *testing.T) {
	h := newHarness(t, "visibility")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	rec := h.do("PUT", "/api/listing/me/visibility", map[string]any{"published": false})
	mustStatus(t, rec, http.StatusOK)
	var off listingResp
	decode(t, rec, &off)
	if off.Published {
		t.Error("published = true setelah nonaktif, mau false")
	}

	rec = h.do("PUT", "/api/listing/me/visibility", map[string]any{"published": true})
	mustStatus(t, rec, http.StatusOK)
	var on listingResp
	decode(t, rec, &on)
	if !on.Published {
		t.Error("published = false setelah aktifkan kembali, mau true")
	}
}

// TestCreateListing_HorizonAwalMinimal13Periode_FR088 proves a fresh listing
// seeds a calendar of at least 13 weekly periods.
func TestCreateListing_HorizonAwalMinimal13Periode_FR088(t *testing.T) {
	h := newHarness(t, "horizon_13")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	var count int
	if err := h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM availability_period WHERE listing_id = $1`, h.listingRow(t)).Scan(&count); err != nil {
		t.Fatalf("count periods: %v", err)
	}
	if count < 13 {
		t.Errorf("jumlah periode awal = %d, mau minimal 13", count)
	}
}

// TestCreateListing_HorizonAwalMinimal3Bulan_FR017 proves the initial calendar
// reaches at least three months (13 weeks) into the future.
func TestCreateListing_HorizonAwalMinimal3Bulan_FR017(t *testing.T) {
	h := newHarness(t, "horizon_3bulan")
	rec := h.do("POST", "/api/listing/me", h.validCreate(500))
	mustStatus(t, rec, http.StatusCreated)
	var got listingResp
	decode(t, rec, &got)

	horizon, err := time.Parse(dateFmt, got.HorizonUntil)
	if err != nil {
		t.Fatalf("parse horizon %q: %v", got.HorizonUntil, err)
	}
	minWeek := mondayAfter(13)
	if horizon.Before(minWeek) {
		t.Errorf("horizon_until = %s, mau minimal %s", got.HorizonUntil, minWeek.Format(dateFmt))
	}
}

// TestCreateListing_SetiapWeekStartHariSenin_FR017 proves every seeded period's
// week_start is a Monday.
func TestCreateListing_SetiapWeekStartHariSenin_FR017(t *testing.T) {
	h := newHarness(t, "horizon_senin")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	rows, err := h.pool.Query(context.Background(),
		`SELECT week_start FROM availability_period WHERE listing_id = $1`, h.listingRow(t))
	if err != nil {
		t.Fatalf("query periods: %v", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var wk time.Time
		if err := rows.Scan(&wk); err != nil {
			t.Fatalf("scan week: %v", err)
		}
		if wk.Weekday() != time.Monday {
			t.Errorf("week_start %s bukan Senin (%s)", wk.Format(dateFmt), wk.Weekday())
		}
		n++
	}
	if n == 0 {
		t.Fatal("tidak ada periode terseed")
	}
}

// TestEnsureHorizon_HorizonUntilKonsistenDenganMaxWeekStart_FR088 proves the
// stored horizon_until equals the furthest period week.
func TestEnsureHorizon_HorizonUntilKonsistenDenganMaxWeekStart_FR088(t *testing.T) {
	h := newHarness(t, "horizon_konsisten")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)
	listingID := h.listingRow(t)

	var horizon, maxWeek time.Time
	if err := h.pool.QueryRow(context.Background(),
		`SELECT horizon_until FROM capacity_listing WHERE id = $1`, listingID).Scan(&horizon); err != nil {
		t.Fatalf("read horizon: %v", err)
	}
	if err := h.pool.QueryRow(context.Background(),
		`SELECT max(week_start) FROM availability_period WHERE listing_id = $1`, listingID).Scan(&maxWeek); err != nil {
		t.Fatalf("read max week: %v", err)
	}
	if !horizon.Equal(maxWeek) {
		t.Errorf("horizon_until = %s, max week_start = %s, mau sama", horizon.Format(dateFmt), maxWeek.Format(dateFmt))
	}
}

// TestEnsureHorizon_DipanggilUlang_TidakMenggandakanBaris_FR088 proves a second
// call to the same week is idempotent: no duplicate period rows.
func TestEnsureHorizon_DipanggilUlang_TidakMenggandakanBaris_FR088(t *testing.T) {
	h := newHarness(t, "horizon_idempoten")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)
	listingID := h.listingRow(t)

	var before int
	if err := h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM availability_period WHERE listing_id = $1`, listingID).Scan(&before); err != nil {
		t.Fatalf("count before: %v", err)
	}

	// Re-run EnsureHorizon to the same week it already covers.
	if err := db.WithTx(context.Background(), h.pool, func(tx pgx.Tx) error {
		_, err := h.svc.EnsureHorizon(context.Background(), tx, listingID, mondayAfter(InitialHorizonWeeks))
		return err
	}); err != nil {
		t.Fatalf("ensure horizon ulang: %v", err)
	}

	var after int
	if err := h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM availability_period WHERE listing_id = $1`, listingID).Scan(&after); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if before != after {
		t.Errorf("jumlah periode berubah dari %d ke %d, mau tetap", before, after)
	}
}

// TestEnsureHorizon_TidakPernahMemundurkanHorizon_FR088 proves calling with an
// earlier week leaves horizon_until untouched.
func TestEnsureHorizon_TidakPernahMemundurkanHorizon_FR088(t *testing.T) {
	h := newHarness(t, "horizon_mundur")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)
	listingID := h.listingRow(t)

	var before time.Time
	if err := h.pool.QueryRow(context.Background(),
		`SELECT horizon_until FROM capacity_listing WHERE id = $1`, listingID).Scan(&before); err != nil {
		t.Fatalf("read horizon before: %v", err)
	}

	if err := db.WithTx(context.Background(), h.pool, func(tx pgx.Tx) error {
		_, err := h.svc.EnsureHorizon(context.Background(), tx, listingID, mondayAfter(2))
		return err
	}); err != nil {
		t.Fatalf("ensure horizon lebih dekat: %v", err)
	}

	var after time.Time
	if err := h.pool.QueryRow(context.Background(),
		`SELECT horizon_until FROM capacity_listing WHERE id = $1`, listingID).Scan(&after); err != nil {
		t.Fatalf("read horizon after: %v", err)
	}
	if !after.Equal(before) {
		t.Errorf("horizon mundur dari %s ke %s", before.Format(dateFmt), after.Format(dateFmt))
	}
}

// TestEnsureHorizon_TanggalBukanSenin_DibulatkanKeSenin_FR017 proves a midweek
// untilWeek is snapped to its Monday before any period is created.
func TestEnsureHorizon_TanggalBukanSenin_DibulatkanKeSenin_FR017(t *testing.T) {
	h := newHarness(t, "horizon_bukansenin")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)
	listingID := h.listingRow(t)

	// A Thursday well past the initial horizon.
	thursday := mondayAfter(20).AddDate(0, 0, 3)
	if thursday.Weekday() != time.Thursday {
		t.Fatalf("tanggal uji %s bukan Kamis", thursday.Weekday())
	}
	var horizon time.Time
	if err := db.WithTx(context.Background(), h.pool, func(tx pgx.Tx) error {
		got, err := h.svc.EnsureHorizon(context.Background(), tx, listingID, thursday)
		horizon = got
		return err
	}); err != nil {
		t.Fatalf("ensure horizon: %v", err)
	}
	if horizon.Weekday() != time.Monday {
		t.Errorf("horizon %s bukan Senin", horizon.Format(dateFmt))
	}
	if !horizon.Equal(mondayAfter(20)) {
		t.Errorf("horizon = %s, mau %s (Senin dari Kamis)", horizon.Format(dateFmt), mondayAfter(20).Format(dateFmt))
	}
}

// TestEnsureHorizon_TidakMengubahCalendarUpdatedAt_FR021 proves extending the
// horizon never stamps calendar_updated_at: only owner edits do (FR-021).
func TestEnsureHorizon_TidakMengubahCalendarUpdatedAt_FR021(t *testing.T) {
	h := newHarness(t, "horizon_calupd")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)
	listingID := h.listingRow(t)

	var before time.Time
	if err := h.pool.QueryRow(context.Background(),
		`SELECT calendar_updated_at FROM capacity_listing WHERE id = $1`, listingID).Scan(&before); err != nil {
		t.Fatalf("read calendar_updated_at: %v", err)
	}

	if err := db.WithTx(context.Background(), h.pool, func(tx pgx.Tx) error {
		_, err := h.svc.EnsureHorizon(context.Background(), tx, listingID, mondayAfter(30))
		return err
	}); err != nil {
		t.Fatalf("ensure horizon: %v", err)
	}

	var after time.Time
	if err := h.pool.QueryRow(context.Background(),
		`SELECT calendar_updated_at FROM capacity_listing WHERE id = $1`, listingID).Scan(&after); err != nil {
		t.Fatalf("read calendar_updated_at after: %v", err)
	}
	if !after.Equal(before) {
		t.Errorf("calendar_updated_at berubah dari %s ke %s saat perpanjang horizon", before, after)
	}
}

// TestUpdatePeriods_MenandaiMingguPenuh_FR017 proves marking a free week full
// zeroes its remaining and sets marked_full.
func TestUpdatePeriods_MenandaiMingguPenuh_FR017(t *testing.T) {
	h := newHarness(t, "periods_penuh")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	week := mondayAfter(4)
	rec := h.do("PUT", "/api/listing/me/periods", map[string]any{
		"periods": []map[string]any{
			{"week_start": week.Format(dateFmt), "capacity": 500, "marked_full": true},
		},
	})
	mustStatus(t, rec, http.StatusOK)
	var got []periodResp
	decode(t, rec, &got)
	if len(got) != 1 {
		t.Fatalf("periode dikembalikan = %d, mau 1", len(got))
	}
	if !got[0].MarkedFull {
		t.Error("marked_full = false, mau true")
	}
	if got[0].Remaining != 0 {
		t.Errorf("remaining = %d, mau 0 saat penuh", got[0].Remaining)
	}
}

// TestUpdatePeriods_KapasitasDiBawahYangTerpakai_Conflict_FR089 proves lowering
// a period's capacity below its used capacity is a 409.
func TestUpdatePeriods_KapasitasDiBawahYangTerpakai_Conflict_FR089(t *testing.T) {
	h := newHarness(t, "periods_konflik")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	week := mondayAfter(3)
	h.seedAllocation(t, h.listingRow(t), h.profileID, week, 400)

	rec := h.do("PUT", "/api/listing/me/periods", map[string]any{
		"periods": []map[string]any{
			{"week_start": week.Format(dateFmt), "capacity": 300, "marked_full": false},
		},
	})
	mustStatus(t, rec, http.StatusConflict)
	if p := problemBody(t, rec); p.Code != "CAPACITY_ALREADY_ALLOCATED" {
		t.Errorf("code = %s, mau CAPACITY_ALREADY_ALLOCATED", p.Code)
	}
}

// TestUpdatePeriods_TandaiPenuhSaatAdaAlokasi_Conflict_FR017 proves marking a
// week full while it carries a live allocation is a 409.
func TestUpdatePeriods_TandaiPenuhSaatAdaAlokasi_Conflict_FR017(t *testing.T) {
	h := newHarness(t, "periods_penuhalokasi")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	week := mondayAfter(3)
	h.seedAllocation(t, h.listingRow(t), h.profileID, week, 200)

	rec := h.do("PUT", "/api/listing/me/periods", map[string]any{
		"periods": []map[string]any{
			{"week_start": week.Format(dateFmt), "capacity": 500, "marked_full": true},
		},
	})
	mustStatus(t, rec, http.StatusConflict)
	if p := problemBody(t, rec); p.Code != "PERIOD_ALREADY_ALLOCATED" {
		t.Errorf("code = %s, mau PERIOD_ALREADY_ALLOCATED", p.Code)
	}
}

// TestUpdatePeriods_SatuElemenGagal_TidakAdaYangTertulis_FR017 proves a batch
// with one conflicting element writes nothing: the transaction rolls back whole.
func TestUpdatePeriods_SatuElemenGagal_TidakAdaYangTertulis_FR017(t *testing.T) {
	h := newHarness(t, "periods_atomik")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	conflictWeek := mondayAfter(3)
	h.seedAllocation(t, h.listingRow(t), h.profileID, conflictWeek, 400)
	goodWeek := mondayAfter(2)

	rec := h.do("PUT", "/api/listing/me/periods", map[string]any{
		"periods": []map[string]any{
			{"week_start": goodWeek.Format(dateFmt), "capacity": 120, "marked_full": false},
			{"week_start": conflictWeek.Format(dateFmt), "capacity": 300, "marked_full": false},
		},
	})
	mustStatus(t, rec, http.StatusConflict)

	// The good week must NOT have been written to 120: whole batch rolled back.
	var cap int32
	if err := h.pool.QueryRow(context.Background(),
		`SELECT total_capacity FROM availability_period WHERE listing_id = $1 AND week_start = $2`,
		h.listingRow(t), goodWeek).Scan(&cap); err != nil {
		t.Fatalf("lookup good week: %v", err)
	}
	if cap == 120 {
		t.Error("minggu bebas tertulis 120 walau batch gagal, mau tidak tersentuh")
	}
}

// TestUpdatePeriods_MingguDiLuarHorizon_HorizonDiperpanjang_FR088 proves editing
// a week beyond the current horizon extends the calendar to cover it.
func TestUpdatePeriods_MingguDiLuarHorizon_HorizonDiperpanjang_FR088(t *testing.T) {
	h := newHarness(t, "periods_perpanjang")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	far := mondayAfter(20)
	rec := h.do("PUT", "/api/listing/me/periods", map[string]any{
		"periods": []map[string]any{
			{"week_start": far.Format(dateFmt), "capacity": 400, "marked_full": false},
		},
	})
	mustStatus(t, rec, http.StatusOK)

	var horizon time.Time
	if err := h.pool.QueryRow(context.Background(),
		`SELECT horizon_until FROM capacity_listing WHERE id = $1`, h.listingRow(t)).Scan(&horizon); err != nil {
		t.Fatalf("read horizon: %v", err)
	}
	if horizon.Before(far) {
		t.Errorf("horizon_until = %s, mau minimal %s", horizon.Format(dateFmt), far.Format(dateFmt))
	}
}

// TestUpdatePeriods_MemperbaruiCalendarUpdatedAt_FR021 proves a successful
// periods edit stamps calendar_updated_at, the sole path that advances it.
func TestUpdatePeriods_MemperbaruiCalendarUpdatedAt_FR021(t *testing.T) {
	h := newHarness(t, "periods_calupd")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)
	listingID := h.listingRow(t)

	var before time.Time
	if err := h.pool.QueryRow(context.Background(),
		`SELECT calendar_updated_at FROM capacity_listing WHERE id = $1`, listingID).Scan(&before); err != nil {
		t.Fatalf("read calendar_updated_at: %v", err)
	}

	// Advance the clock so a fresh stamp is distinguishable from the create stamp.
	h.clock.Set(baseTime.Add(2 * time.Hour))
	week := mondayAfter(5)
	mustStatus(t, h.do("PUT", "/api/listing/me/periods", map[string]any{
		"periods": []map[string]any{
			{"week_start": week.Format(dateFmt), "capacity": 400, "marked_full": false},
		},
	}), http.StatusOK)

	var after time.Time
	if err := h.pool.QueryRow(context.Background(),
		`SELECT calendar_updated_at FROM capacity_listing WHERE id = $1`, listingID).Scan(&after); err != nil {
		t.Fatalf("read calendar_updated_at after: %v", err)
	}
	if !after.After(before) {
		t.Errorf("calendar_updated_at = %s, mau lebih baru dari %s", after, before)
	}
}

// TestUpdatePeriods_WeekStartBukanSenin_Ditolak_FR017 proves a non-Monday
// week_start is a 422 naming week_start.
func TestUpdatePeriods_WeekStartBukanSenin_Ditolak_FR017(t *testing.T) {
	h := newHarness(t, "periods_bukansenin")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)

	tuesday := mondayAfter(3).AddDate(0, 0, 1)
	rec := h.do("PUT", "/api/listing/me/periods", map[string]any{
		"periods": []map[string]any{
			{"week_start": tuesday.Format(dateFmt), "capacity": 400, "marked_full": false},
		},
	})
	mustStatus(t, rec, http.StatusUnprocessableEntity)
	p := problemBody(t, rec)
	found := false
	for _, e := range p.Errors {
		if e.Field == "week_start" {
			found = true
		}
	}
	if !found {
		t.Errorf("errors = %+v, mau menyebut week_start", p.Errors)
	}
}

// TestUpdatePeriods_TanpaPeranSubkontraktor_Forbidden_FR017 proves a caller
// without the subcontractor role is rejected before the handler runs.
func TestUpdatePeriods_TanpaPeranSubkontraktor_Forbidden_FR017(t *testing.T) {
	h := newHarness(t, "periods_peran")
	h.auth.principal = &httpx.Principal{
		Roles:   httpx.RoleBuyer,
		Account: sqlcgen.UserAccount{ID: h.accountID},
	}
	rec := h.do("PUT", "/api/listing/me/periods", map[string]any{
		"periods": []map[string]any{
			{"week_start": mondayAfter(3).Format(dateFmt), "capacity": 400, "marked_full": false},
		},
	})
	mustStatus(t, rec, http.StatusForbidden)
}

// TestListingRoutes_TidakMeninggalkanRuteTanpaKeputusanPeran proves every /api
// route the listing package registers sits behind a role gate, so the coverage
// gate cannot pass while a listing endpoint ships without a role decision. It
// runs without a database: Register only records patterns.
func TestListingRoutes_TidakMeninggalkanRuteTanpaKeputusanPeran(t *testing.T) {
	svc := &Service{}
	r := httpx.NewRouter(quietLogger())
	svc.Register(r, &mockAuth{})

	if got := r.UncoveredAPIRoutes(); len(got) != 0 {
		t.Fatalf("rute listing tak tercakup = %v, mau kosong", got)
	}
}
