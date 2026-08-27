package reputation

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// baseTime is a fixed Monday (2026-08-24), so WeekStart(now) is the day itself
// and every readiness week seeded below satisfies readiness_is_monday.
var baseTime = time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)

// phoneSeq hands each seeded account a unique phone matching the user_account
// phone_format check (^62[0-9]{8,13}$).
var phoneSeq atomic.Int64

func nextPhone() string {
	return "628" + strconv.Itoa(300000000+int(phoneSeq.Add(1)))
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// mockAuth is a stand-in Authenticator so these tests never import the account
// package, which would invert the dependency direction this package keeps
// (account -> reputation, never back). A nil principal is unauthenticated.
type mockAuth struct {
	principal *httpx.Principal
}

func (m *mockAuth) Authenticate(_ *http.Request) (httpx.Principal, error) {
	if m.principal == nil {
		return httpx.Principal{}, httpx.ErrUnauthenticated
	}
	return *m.principal, nil
}

// party is one seeded business: its account id and its profile id.
type party struct {
	account pgtype.UUID
	profile pgtype.UUID
}

// harness owns a migrated schema, the reputation service under test driven by a
// TestClock, a router carrying its two routes, and one buyer plus one
// subcontractor to transact between.
type harness struct {
	svc       *Service
	handler   http.Handler
	pool      *pgxpool.Pool
	clock     *platform.TestClock
	auth      *mockAuth
	buyer     party
	subcon    party
	listingID pgtype.UUID
	productID pgtype.UUID
}

func newHarness(t *testing.T, name string) *harness {
	t.Helper()
	pool := testdb.New(t, name)
	clock := platform.NewTestClock(baseTime)
	svc := New(pool, clock)

	seedRegion(t, pool)
	productID := seedCatalogItem(t, pool, "product", "Kaos Oblong")

	buyer := seedParty(t, pool, "pembeli-"+name, false)
	subcon := seedParty(t, pool, "subkon-"+name, true)

	auth := &mockAuth{principal: buyerPrincipal(buyer.account)}
	r := httpx.NewRouter(quietLogger())
	svc.Register(r, auth)

	h := &harness{
		svc: svc, handler: r.Handler(), pool: pool, clock: clock, auth: auth,
		buyer: buyer, subcon: subcon, productID: productID,
	}
	h.listingID = seedListing(t, h, subcon.profile)
	return h
}

func buyerPrincipal(account pgtype.UUID) *httpx.Principal {
	return &httpx.Principal{
		Roles:   httpx.RoleBuyer,
		Account: sqlcgen.UserAccount{ID: account},
	}
}

func subconPrincipal(account pgtype.UUID) *httpx.Principal {
	return &httpx.Principal{
		Roles:   httpx.RoleSubcontractor,
		Account: sqlcgen.UserAccount{ID: account},
	}
}

// adminPrincipal is the admin caller the T069 moderation route requires. The
// account need not be a seeded row: the route reads only the id off the
// principal to record hidden_by, and hidden_by references user_account, so a
// hide test seeds a real admin account and passes its id here.
func adminPrincipal(account pgtype.UUID) *httpx.Principal {
	return &httpx.Principal{
		Roles:   httpx.RoleAdmin,
		Account: sqlcgen.UserAccount{ID: account},
	}
}

// seedAdmin inserts a bare admin account and returns its id, for hidden_by which
// references user_account.
func seedAdmin(t *testing.T, pool *pgxpool.Pool, name string) pgtype.UUID {
	t.Helper()
	var acc pgtype.UUID
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO user_account (email, phone, password_hash, role_admin, created_at, updated_at)
		 VALUES ($1, $2, 'x', true, $3, $3) RETURNING id`,
		name+"@contoh.test", nextPhone(), baseTime).Scan(&acc); err != nil {
		t.Fatalf("seed admin %s: %v", name, err)
	}
	return acc
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

func seedCatalogItem(t *testing.T, pool *pgxpool.Pool, kind sqlcgen.ItemType, name string) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO catalog_item (type, name, active, created_at) VALUES ($1, $2, true, $3) RETURNING id`,
		kind, name, baseTime).Scan(&id); err != nil {
		t.Fatalf("seed catalog item: %v", err)
	}
	return id
}

// seedParty creates an account and its business profile in one step, the pair a
// review needs on both sides (the review columns hold profile ids, the session
// carries the account id).
func seedParty(t *testing.T, pool *pgxpool.Pool, name string, subcontractor bool) party {
	t.Helper()
	ctx := context.Background()
	var acc pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO user_account (email, phone, password_hash, role_subcontractor, role_buyer, created_at, updated_at)
		 VALUES ($1, $2, 'x', $3, $4, $5, $5) RETURNING id`,
		name+"@contoh.test", nextPhone(), subcontractor, !subcontractor, baseTime).Scan(&acc); err != nil {
		t.Fatalf("seed account %s: %v", name, err)
	}
	var prof pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO business_profile (account_id, business_name, city_code, verified, created_at, updated_at)
		 VALUES ($1, $2, '3273', true, $3, $3) RETURNING id`,
		acc, "Usaha "+name, baseTime).Scan(&prof); err != nil {
		t.Fatalf("seed profile %s: %v", name, err)
	}
	return party{account: acc, profile: prof}
}

func seedListing(t *testing.T, h *harness, ownerProfile pgtype.UUID) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	weekNow := platform.WeekStart(baseTime)
	var listingID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO capacity_listing (profile_id, weekly_capacity, readiness_lead_days, published, calendar_updated_at, horizon_until, created_at, updated_at)
		 VALUES ($1, 500, 0, true, $2, $3, $2, $2) RETURNING id`,
		ownerProfile, baseTime, weekNow.AddDate(0, 0, 7*8)).Scan(&listingID); err != nil {
		t.Fatalf("seed listing: %v", err)
	}
	if _, err := h.pool.Exec(ctx,
		`INSERT INTO listing_product (listing_id, item_id) VALUES ($1, $2)`,
		listingID, h.productID); err != nil {
		t.Fatalf("seed listing_product: %v", err)
	}
	return listingID
}

// orderSpec describes one seeded work order. status drives which of the
// timestamp columns the constraints require: a cancelled order needs all three
// cancellation columns, a shipped order needs shipped_at.
type orderSpec struct {
	buyer       party
	subcon      party
	status      string
	shippedAt   *time.Time
	confirmedAt *time.Time
	cancelledBy *pgtype.UUID
}

// seedOrder builds the full quota_request -> request_candidate -> offer ->
// work_order chain. GetWorkOrderForView joins every one of those tables, so a
// bare work_order row would not be readable by the handler under test.
func seedOrder(t *testing.T, h *harness, spec orderSpec) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	weekNow := platform.WeekStart(baseTime)
	deadline := weekNow.AddDate(0, 0, 7*4)

	var requestID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO quota_request (buyer_id, product_item_id, quantity, material, deadline, reply_due_at, created_at)
		 VALUES ($1, $2, 100, 'Katun', $3, $4, $5) RETURNING id`,
		spec.buyer.profile, h.productID, deadline, baseTime.Add(48*time.Hour), baseTime).Scan(&requestID); err != nil {
		t.Fatalf("seed quota_request: %v", err)
	}

	var candidateID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO request_candidate (request_id, listing_id, subcontractor_id, status, updated_at)
		 VALUES ($1, $2, $3, 'agreed', $4) RETURNING id`,
		requestID, h.listingID, spec.subcon.profile, baseTime).Scan(&candidateID); err != nil {
		t.Fatalf("seed request_candidate: %v", err)
	}

	var offerID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO offer (candidate_id, sequence, proposed_by, total_price, readiness_lead_days, created_at)
		 VALUES ($1, 1, 'subcontractor', 100000, 0, $2) RETURNING id`,
		candidateID, baseTime).Scan(&offerID); err != nil {
		t.Fatalf("seed offer: %v", err)
	}

	var cancelledBy any
	var cancelReason any
	var cancelledAt any
	if spec.status == "cancelled" {
		by := spec.subcon.profile
		if spec.cancelledBy != nil {
			by = *spec.cancelledBy
		}
		cancelledBy = by
		cancelReason = "Kapasitas tidak tersedia."
		cancelledAt = baseTime
	}

	var orderID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO work_order (candidate_id, offer_id, buyer_id, subcontractor_id, quantity, total_price,
		     deadline, readiness_week_start, status, shipped_at, confirmed_at,
		     cancelled_by_id, cancellation_reason, cancelled_at, created_at)
		 VALUES ($1, $2, $3, $4, 100, 100000, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id`,
		candidateID, offerID, spec.buyer.profile, spec.subcon.profile, deadline, weekNow, spec.status,
		spec.shippedAt, spec.confirmedAt, cancelledBy, cancelReason, cancelledAt, baseTime).Scan(&orderID); err != nil {
		t.Fatalf("seed work_order (%s): %v", spec.status, err)
	}
	return orderID
}

// seedConfirmedOrder is the common case: an order both parties may review.
func seedConfirmedOrder(t *testing.T, h *harness) pgtype.UUID {
	t.Helper()
	at := baseTime
	return seedOrder(t, h, orderSpec{
		buyer: h.buyer, subcon: h.subcon, status: "confirmed", confirmedAt: &at,
	})
}

// viewOf runs the one SearchReputation query both consumers run and derives the
// block from it, so a test asserts on exactly what the profile and the search
// would render.
func (h *harness) viewOf(t *testing.T, profileID pgtype.UUID) View {
	t.Helper()
	rows, err := h.svc.queries().SearchReputation(context.Background(), []pgtype.UUID{profileID})
	if err != nil {
		t.Fatalf("SearchReputation: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("mau 1 baris reputasi, dapat %d", len(rows))
	}
	r := rows[0]
	var avg *float64
	if f, err := r.AverageRating.Float64Value(); err == nil && f.Valid {
		v := f.Float64
		avg = &v
	}
	return Derive(int(r.CompletionCompleted), int(r.CompletionDivisor), int(r.ReviewCount), avg)
}

func (h *harness) do(method, path string, body any) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.7:9999"
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

func (h *harness) postReview(orderID pgtype.UUID, body any) *httptest.ResponseRecorder {
	return h.do(http.MethodPost, "/api/work-orders/"+uuidString(orderID)+"/reviews", body)
}

func mustStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status %d, mau %d; body %s", rec.Code, want, rec.Body.String())
	}
}

type problemResp struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) problemResp {
	t.Helper()
	var p problemResp
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem %q: %v", rec.Body.String(), err)
	}
	return p
}

func decodeReview(t *testing.T, rec *httptest.ResponseRecorder) reviewView {
	t.Helper()
	var v reviewView
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode review %q: %v", rec.Body.String(), err)
	}
	return v
}

func decodeList(t *testing.T, rec *httptest.ResponseRecorder) reviewList {
	t.Helper()
	var v reviewList
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode list %q: %v", rec.Body.String(), err)
	}
	return v
}
