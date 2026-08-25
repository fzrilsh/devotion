package quota

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
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
// and reply_due_at = now + 72h is deterministic.
var baseTime = time.Date(2026, 8, 24, 10, 3, 0, 0, time.UTC)

const dateFmt = "2006-01-02"

// phoneSeq hands each seeded account a unique phone matching the phone_format
// check (^62[0-9]{8,13}$).
var phoneSeq atomic.Int64

func nextPhone() string {
	return "628" + strconv.Itoa(100000000+int(phoneSeq.Add(1)))
}

// mockAuth is a stand-in Authenticator so the quota tests never import account.
type mockAuth struct {
	principal *httpx.Principal
}

func (m *mockAuth) Authenticate(_ *http.Request) (httpx.Principal, error) {
	if m.principal == nil {
		return httpx.Principal{}, httpx.ErrUnauthenticated
	}
	return *m.principal, nil
}

// recordNotifier records the account ids notified without touching a queue.
type recordNotifier struct {
	events []sqlcgen.EventType
}

func (n *recordNotifier) Enqueue(_ context.Context, _ pgx.Tx, _ pgtype.UUID, event sqlcgen.EventType, _, _ string, _ *string) error {
	n.events = append(n.events, event)
	return nil
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type harness struct {
	svc       *Service
	handler   http.Handler
	pool      *pgxpool.Pool
	clock     *platform.TestClock
	auth      *mockAuth
	notifier  *recordNotifier
	buyerAcc  pgtype.UUID
	buyerProf pgtype.UUID
	productID pgtype.UUID
	machineID pgtype.UUID
}

func newHarness(t *testing.T, name string) *harness {
	t.Helper()
	pool := testdb.New(t, name)
	clock := platform.NewTestClock(baseTime)
	notifier := &recordNotifier{}
	svc := New(pool, clock, notifier)

	seedRegion(t, pool)
	buyerAcc := seedAccount(t, pool, "buyer@contoh.test", "628110000009", false)
	buyerProf := seedProfile(t, pool, buyerAcc, "Merek Pembeli", "3273")

	auth := &mockAuth{principal: &httpx.Principal{
		Roles:   httpx.RoleBuyer,
		Account: sqlcgen.UserAccount{ID: buyerAcc},
	}}

	r := httpx.NewRouter(quietLogger())
	svc.Register(r, auth)

	h := &harness{
		svc: svc, handler: r.Handler(), pool: pool, clock: clock,
		auth: auth, notifier: notifier, buyerAcc: buyerAcc, buyerProf: buyerProf,
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

func seedProfile(t *testing.T, pool *pgxpool.Pool, accountID pgtype.UUID, name, cityCode string) pgtype.UUID {
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

// seedListing creates a published listing owned by a fresh subcontractor and
// returns the listing id and the owning profile id.
func seedListing(t *testing.T, h *harness, name string) (pgtype.UUID, pgtype.UUID) {
	t.Helper()
	ctx := context.Background()
	pool := h.pool

	acc := seedAccount(t, pool, "subkon-"+name+"@contoh.test", nextPhone(), true)
	prof := seedProfile(t, pool, acc, name, "3273")

	weekNow := platform.WeekStart(baseTime)
	horizon := weekNow.AddDate(0, 0, 7*8)
	var listingID pgtype.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO capacity_listing (profile_id, weekly_capacity, readiness_lead_days, published, calendar_updated_at, horizon_until, created_at, updated_at)
		 VALUES ($1, 100, 7, true, $2, $3, $2, $2) RETURNING id`,
		prof, baseTime, horizon).Scan(&listingID)
	if err != nil {
		t.Fatalf("seed listing: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO listing_product (listing_id, item_id) VALUES ($1, $2)`,
		listingID, h.productID); err != nil {
		t.Fatalf("seed listing_product: %v", err)
	}
	for w := weekNow; !w.After(horizon); w = w.AddDate(0, 0, 7) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO availability_period (listing_id, week_start, total_capacity, used_capacity, created_at, updated_at)
			 VALUES ($1, $2, 100, 0, $3, $3)`,
			listingID, w, baseTime); err != nil {
			t.Fatalf("seed period %s: %v", w.Format(dateFmt), err)
		}
	}
	return listingID, prof
}

// seedOwnListing creates a published listing owned by the buyer's own profile,
// the self-request path FR-083 must reject.
func seedOwnListing(t *testing.T, h *harness) pgtype.UUID {
	t.Helper()
	horizon := platform.WeekStart(baseTime).AddDate(0, 0, 7*8)
	var listingID pgtype.UUID
	err := h.pool.QueryRow(context.Background(),
		`INSERT INTO capacity_listing (profile_id, weekly_capacity, readiness_lead_days, published, calendar_updated_at, horizon_until, created_at, updated_at)
		 VALUES ($1, 100, 7, true, $2, $3, $2, $2) RETURNING id`,
		h.buyerProf, baseTime, horizon).Scan(&listingID)
	if err != nil {
		t.Fatalf("seed own listing: %v", err)
	}
	if _, err := h.pool.Exec(context.Background(),
		`INSERT INTO listing_product (listing_id, item_id) VALUES ($1, $2)`,
		listingID, h.productID); err != nil {
		t.Fatalf("seed own listing_product: %v", err)
	}
	return listingID
}

func (h *harness) doJSON(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.7:9999"
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
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
	CandidateID  string `json:"candidate_id"`
	ListingID    string `json:"listing_id"`
	ProfileID    string `json:"profile_id"`
	BusinessName string `json:"business_name"`
	Status       string `json:"status"`
}

type requestResp struct {
	RequestID     string          `json:"request_id"`
	ProductItemID string          `json:"product_item_id"`
	Quantity      int             `json:"quantity"`
	Material      string          `json:"material"`
	Deadline      string          `json:"deadline"`
	Note          *string         `json:"note"`
	Candidates    []candidateResp `json:"candidates"`
	CreatedAt     time.Time       `json:"created_at"`
	ExpiresAt     time.Time       `json:"expires_at"`
}

type listResp struct {
	Items      []requestResp `json:"items"`
	Pagination struct {
		HasNext    bool    `json:"has_next"`
		NextCursor *string `json:"next_cursor"`
	} `json:"pagination"`
}

type problemResp struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Errors []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"errors"`
}

func decodeRequest(t *testing.T, rec *httptest.ResponseRecorder) requestResp {
	t.Helper()
	var r requestResp
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode request %q: %v", rec.Body.String(), err)
	}
	return r
}

func decodeList(t *testing.T, rec *httptest.ResponseRecorder) listResp {
	t.Helper()
	var r listResp
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode list %q: %v", rec.Body.String(), err)
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

func deadlineParam(n int) string {
	return platform.WeekStart(baseTime).AddDate(0, 0, 7*n).Format(dateFmt)
}

// maxUUID is a syntactically valid UUID that no seeded row uses, so the service
// treats it as an unknown listing (422) rather than a malformed id.
func maxUUID() pgtype.UUID {
	var u pgtype.UUID
	for i := range u.Bytes {
		u.Bytes[i] = 0xff
	}
	u.Valid = true
	return u
}

// createBody builds a QuotaRequestCreate JSON body for the given listings.
func (h *harness) createBody(listingIDs []pgtype.UUID, quantity int) string {
	ids := make([]string, len(listingIDs))
	for i, id := range listingIDs {
		ids[i] = uuidString(id)
	}
	b, _ := json.Marshal(map[string]any{
		"listing_ids":     ids,
		"product_item_id": uuidString(h.productID),
		"quantity":        quantity,
		"material":        "Katun combed 30s",
		"deadline":        deadlineParam(4),
		"note":            "Mohon rapikan jahitan.",
	})
	return string(b)
}

// TestQuotaRequest_HappyPath_SendsToSeveralCandidates_FR029_FR030 sends one
// request to three candidates in a single action and asserts each comes back
// with status awaiting_reply and its own candidate row.
func TestQuotaRequest_HappyPath_SendsToSeveralCandidates_FR029_FR030(t *testing.T) {
	h := newHarness(t, "quota_happy")
	l1, p1 := seedListing(t, h, "alfa")
	l2, p2 := seedListing(t, h, "beta")
	l3, p3 := seedListing(t, h, "gama")

	rec := h.doJSON(http.MethodPost, "/api/quota-requests", h.createBody([]pgtype.UUID{l1, l2, l3}, 50))
	mustStatus(t, rec, http.StatusCreated)

	res := decodeRequest(t, rec)
	if res.RequestID == "" {
		t.Fatalf("request_id kosong; body %s", rec.Body.String())
	}
	if len(res.Candidates) != 3 {
		t.Fatalf("mau 3 kandidat, dapat %d", len(res.Candidates))
	}
	wantProfiles := map[string]bool{uuidString(p1): true, uuidString(p2): true, uuidString(p3): true}
	for _, c := range res.Candidates {
		if c.Status != "awaiting_reply" {
			t.Fatalf("status kandidat %q, mau awaiting_reply", c.Status)
		}
		if !wantProfiles[c.ProfileID] {
			t.Fatalf("profile_id kandidat %q tak terduga", c.ProfileID)
		}
	}
	if res.Material != "Katun combed 30s" {
		t.Fatalf("material %q", res.Material)
	}
	if len(h.notifier.events) != 3 {
		t.Fatalf("mau 3 notifikasi, dapat %d", len(h.notifier.events))
	}
}

// TestQuotaRequest_SetsReplyDue72Hours_FR082 proves the system sets the reply
// window to exactly 72 hours from the injected Clock, and the buyer supplies no
// column to control it. It reads reply_due_at straight from the row.
func TestQuotaRequest_SetsReplyDue72Hours_FR082(t *testing.T) {
	h := newHarness(t, "quota_due")
	l1, _ := seedListing(t, h, "alfa")

	rec := h.doJSON(http.MethodPost, "/api/quota-requests", h.createBody([]pgtype.UUID{l1}, 50))
	mustStatus(t, rec, http.StatusCreated)
	res := decodeRequest(t, rec)

	wantExpiry := baseTime.Add(72 * time.Hour)
	if !res.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("expires_at %v, mau %v (now + 72 jam)", res.ExpiresAt, wantExpiry)
	}

	var replyDue, createdAt time.Time
	err := h.pool.QueryRow(context.Background(),
		`SELECT reply_due_at, created_at FROM quota_request WHERE id = $1`, res.RequestID).Scan(&replyDue, &createdAt)
	if err != nil {
		t.Fatalf("baca reply_due_at: %v", err)
	}
	if !replyDue.Equal(wantExpiry) {
		t.Fatalf("reply_due_at %v, mau %v", replyDue, wantExpiry)
	}
	if !createdAt.Equal(baseTime) {
		t.Fatalf("created_at %v, mau %v (dari Clock)", createdAt, baseTime)
	}
}

// TestQuotaRequest_RejectsSelfListing_FR083 proves a request that includes the
// buyer's own listing is rejected with 409 SELF_REQUEST, including the direct
// path that does not go through search results, and that nothing is inserted.
func TestQuotaRequest_RejectsSelfListing_FR083(t *testing.T) {
	h := newHarness(t, "quota_self")
	other, _ := seedListing(t, h, "alfa")
	own := seedOwnListing(t, h)

	rec := h.doJSON(http.MethodPost, "/api/quota-requests", h.createBody([]pgtype.UUID{other, own}, 50))
	mustStatus(t, rec, http.StatusConflict)
	p := decodeProblem(t, rec)
	if p.Code != "SELF_REQUEST" {
		t.Fatalf("code %q, mau SELF_REQUEST", p.Code)
	}

	var n int
	if err := h.pool.QueryRow(context.Background(), `SELECT count(*) FROM quota_request`).Scan(&n); err != nil {
		t.Fatalf("hitung request: %v", err)
	}
	if n != 0 {
		t.Fatalf("mau 0 request tersimpan, dapat %d (transaksi harus batal)", n)
	}
}

// TestQuotaRequest_RejectsNonBuyer_FR029 proves the gate rejects a caller
// without the buyer role.
func TestQuotaRequest_RejectsNonBuyer_FR029(t *testing.T) {
	h := newHarness(t, "quota_role")
	l1, _ := seedListing(t, h, "alfa")
	h.auth.principal = &httpx.Principal{
		Roles:   httpx.RoleSubcontractor,
		Account: sqlcgen.UserAccount{ID: h.buyerAcc},
	}
	rec := h.doJSON(http.MethodPost, "/api/quota-requests", h.createBody([]pgtype.UUID{l1}, 50))
	mustStatus(t, rec, http.StatusForbidden)
}

// TestQuotaRequest_RejectsUnauthenticated_FR029 proves a missing session is 401.
func TestQuotaRequest_RejectsUnauthenticated_FR029(t *testing.T) {
	h := newHarness(t, "quota_noauth")
	l1, _ := seedListing(t, h, "alfa")
	h.auth.principal = nil
	rec := h.doJSON(http.MethodPost, "/api/quota-requests", h.createBody([]pgtype.UUID{l1}, 50))
	mustStatus(t, rec, http.StatusUnauthorized)
}

// TestQuotaRequest_RejectsInvalidInput_FR029 covers required-field and format
// validation, each a 422.
func TestQuotaRequest_RejectsInvalidInput_FR029(t *testing.T) {
	h := newHarness(t, "quota_invalid")
	l1, _ := seedListing(t, h, "alfa")
	lid := uuidString(l1)
	pid := uuidString(h.productID)

	cases := map[string]string{
		"no listings":  `{"listing_ids":[],"product_item_id":"` + pid + `","quantity":1,"material":"katun","deadline":"` + deadlineParam(4) + `"}`,
		"no product":   `{"listing_ids":["` + lid + `"],"quantity":1,"material":"katun","deadline":"` + deadlineParam(4) + `"}`,
		"bad quantity": `{"listing_ids":["` + lid + `"],"product_item_id":"` + pid + `","quantity":0,"material":"katun","deadline":"` + deadlineParam(4) + `"}`,
		"no material":  `{"listing_ids":["` + lid + `"],"product_item_id":"` + pid + `","quantity":1,"material":"","deadline":"` + deadlineParam(4) + `"}`,
		"bad deadline": `{"listing_ids":["` + lid + `"],"product_item_id":"` + pid + `","quantity":1,"material":"katun","deadline":"bukan-tanggal"}`,
		"bad listing":  `{"listing_ids":["bukan-uuid"],"product_item_id":"` + pid + `","quantity":1,"material":"katun","deadline":"` + deadlineParam(4) + `"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			rec := h.doJSON(http.MethodPost, "/api/quota-requests", body)
			mustStatus(t, rec, http.StatusUnprocessableEntity)
		})
	}
}

// TestQuotaRequest_RejectsUnknownListing_FR029 proves an unknown or unpublished
// listing id is a 422, not a silent drop.
func TestQuotaRequest_RejectsUnknownListing_FR029(t *testing.T) {
	h := newHarness(t, "quota_unknown")
	body := h.createBody([]pgtype.UUID{maxUUID()}, 50)
	rec := h.doJSON(http.MethodPost, "/api/quota-requests", body)
	mustStatus(t, rec, http.StatusUnprocessableEntity)
}

// TestQuotaRequest_ListsOwnRequests_FR030 sends two requests then lists them,
// asserting the buyer sees both newest first.
func TestQuotaRequest_ListsOwnRequests_FR030(t *testing.T) {
	h := newHarness(t, "quota_list")
	l1, _ := seedListing(t, h, "alfa")

	rec := h.doJSON(http.MethodPost, "/api/quota-requests", h.createBody([]pgtype.UUID{l1}, 50))
	mustStatus(t, rec, http.StatusCreated)
	h.clock.Advance(time.Minute)
	rec = h.doJSON(http.MethodPost, "/api/quota-requests", h.createBody([]pgtype.UUID{l1}, 60))
	mustStatus(t, rec, http.StatusCreated)

	rec = h.do(http.MethodGet, "/api/quota-requests")
	mustStatus(t, rec, http.StatusOK)
	res := decodeList(t, rec)
	if len(res.Items) != 2 {
		t.Fatalf("mau 2 request, dapat %d", len(res.Items))
	}
	if res.Items[0].Quantity != 60 {
		t.Fatalf("urutan salah: item pertama quantity %d, mau 60 (terbaru dulu)", res.Items[0].Quantity)
	}
}

// TestQuotaRequest_ListPagination_StableAcrossPages_FR030_FR080 walks the opaque
// cursor and asserts no overlap and no gap.
func TestQuotaRequest_ListPagination_StableAcrossPages_FR030_FR080(t *testing.T) {
	h := newHarness(t, "quota_paging")
	l1, _ := seedListing(t, h, "alfa")
	const total = 5
	for i := 0; i < total; i++ {
		rec := h.doJSON(http.MethodPost, "/api/quota-requests", h.createBody([]pgtype.UUID{l1}, 50+i))
		mustStatus(t, rec, http.StatusCreated)
		h.clock.Advance(time.Minute)
	}

	seen := map[string]bool{}
	path := "/api/quota-requests?size=2"
	for {
		rec := h.do(http.MethodGet, path)
		mustStatus(t, rec, http.StatusOK)
		res := decodeList(t, rec)
		for _, it := range res.Items {
			if seen[it.RequestID] {
				t.Fatalf("request %s muncul dua kali", it.RequestID)
			}
			seen[it.RequestID] = true
		}
		if !res.Pagination.HasNext {
			break
		}
		if res.Pagination.NextCursor == nil {
			t.Fatalf("has_next true tapi next_cursor nil")
		}
		v := url.Values{}
		v.Set("size", "2")
		v.Set("cursor", *res.Pagination.NextCursor)
		path = "/api/quota-requests?" + v.Encode()
	}
	if len(seen) != total {
		t.Fatalf("mau %d request unik, dapat %d", total, len(seen))
	}
}

// TestQuotaRequest_ListRejectsInvalidQuery_FR030 proves the query params of the
// list endpoint are validated as user input: an out-of-range size and a garbage
// cursor are each a 422 with the VALIDATION_FAILED code. FR-030 governs the
// endpoint; no separate FR pins query validation, so what is enforced here is the
// 422 contract response for a malformed page request.
func TestQuotaRequest_ListRejectsInvalidQuery_FR030(t *testing.T) {
	h := newHarness(t, "quota_list_badquery")

	cases := map[string]string{
		"size zero":     "/api/quota-requests?size=0",
		"size over max": "/api/quota-requests?size=51",
		"bad cursor":    "/api/quota-requests?cursor=busuk",
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			rec := h.do(http.MethodGet, path)
			mustStatus(t, rec, http.StatusUnprocessableEntity)
			if p := decodeProblem(t, rec); p.Code != "VALIDATION_FAILED" {
				t.Fatalf("code %q, mau VALIDATION_FAILED", p.Code)
			}
		})
	}
}

// TestQuotaRequest_ListRejectsNonBuyer_FR030 proves the list endpoint is gated to
// the buyer role: a subcontractor principal is rejected before the handler runs.
// The route-coverage test only asserts a role decision exists; this proves the
// GET decision actually rejects the wrong role with 403.
func TestQuotaRequest_ListRejectsNonBuyer_FR030(t *testing.T) {
	h := newHarness(t, "quota_list_role")
	h.auth.principal = &httpx.Principal{
		Roles:   httpx.RoleSubcontractor,
		Account: sqlcgen.UserAccount{ID: h.buyerAcc},
	}
	rec := h.do(http.MethodGet, "/api/quota-requests")
	mustStatus(t, rec, http.StatusForbidden)
}

// TestQuotaRoutes_TidakMeninggalkanRuteTanpaKeputusanPeran proves the quota
// routes sit behind a role gate.
func TestQuotaRoutes_TidakMeninggalkanRuteTanpaKeputusanPeran(t *testing.T) {
	svc := &Service{}
	r := httpx.NewRouter(quietLogger())
	svc.Register(r, &mockAuth{})

	if got := r.UncoveredAPIRoutes(); len(got) != 0 {
		t.Fatalf("rute quota tak tercakup = %v, mau kosong", got)
	}
}
