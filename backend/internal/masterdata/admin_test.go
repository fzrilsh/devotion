package masterdata

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/notification"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// adminHarness wires a masterdata Service against an isolated schema with an
// admin principal, so the T068 admin routes run through the real router and role
// gate. auth starts as an admin; a role test swaps auth.principal.
type adminHarness struct {
	svc       *Service
	handler   http.Handler
	pool      *pgxpool.Pool
	clock     *platform.TestClock
	auth      *mockAuth
	adminID   pgtype.UUID
	buyerID   pgtype.UUID
	buyerProf pgtype.UUID
}

func newAdminHarness(t *testing.T, name string) *adminHarness {
	t.Helper()
	pool := testdb.New(t, name)
	clock := platform.NewTestClock(baseTime)
	notif := notification.New(pool, clock, nil, nil, nil)
	svc := New(pool, clock, nil, notif)

	adminID := seedAccount(t, pool, "admin@contoh.test", "628110000001", false)
	buyerID := seedAccount(t, pool, "buyer@contoh.test", "628110000009", false)
	buyerProf := seedProfile(t, pool, buyerID, "Butik Pemesan")

	auth := &mockAuth{principal: &httpx.Principal{
		Roles:   httpx.RoleAdmin,
		Account: sqlcgen.UserAccount{ID: adminID},
	}}

	r := httpx.NewRouter(quietLogger())
	svc.Register(r, auth)

	return &adminHarness{
		svc: svc, handler: r.Handler(), pool: pool, clock: clock,
		auth: auth, adminID: adminID, buyerID: buyerID, buyerProf: buyerProf,
	}
}

// req drives one request with an optional JSON body and returns the recorder.
func (h *adminHarness) req(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(buf)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

// asBuyer swaps the principal to a buyer, the non-admin role a gate test rejects.
func (h *adminHarness) asBuyer() {
	h.auth.principal = &httpx.Principal{
		Roles:   httpx.RoleBuyer,
		Account: sqlcgen.UserAccount{ID: h.buyerID},
	}
}

// TestAdminListItems_Success_FR059 lists a kind's items, active and inactive, so
// the admin catalog surface reports the full baseline (FR-059).
func TestAdminListItems_Success_FR059(t *testing.T) {
	h := newAdminHarness(t, "admin_list_items")
	seedCatalogItem(t, h.pool, sqlcgen.ItemTypeProduct, "Kaos Oblong")

	rec := h.req(t, http.MethodGet, "/api/admin/master/items?kind=product", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var out []catalogItem
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("urai body: %v", err)
	}
	if len(out) != 1 || out[0].Name != "Kaos Oblong" || out[0].Kind != "product" {
		t.Fatalf("daftar item = %+v, mau satu produk Kaos Oblong", out)
	}
}

// TestAdminListItems_RejectsRole_FR059 confirms the list route is gated to admin:
// a buyer is 403 and an anonymous caller 401.
func TestAdminListItems_RejectsRole_FR059(t *testing.T) {
	h := newAdminHarness(t, "admin_list_role")

	h.asBuyer()
	if rec := h.req(t, http.MethodGet, "/api/admin/master/items?kind=product", nil); rec.Code != http.StatusForbidden {
		t.Errorf("buyer: status = %d, mau 403", rec.Code)
	}
	h.auth.principal = nil
	if rec := h.req(t, http.MethodGet, "/api/admin/master/items?kind=product", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonim: status = %d, mau 401", rec.Code)
	}
}

// TestAdminListItems_RejectsInvalidInput_FR059 rejects a missing or unknown kind
// query param with 422.
func TestAdminListItems_RejectsInvalidInput_FR059(t *testing.T) {
	h := newAdminHarness(t, "admin_list_invalid")

	if rec := h.req(t, http.MethodGet, "/api/admin/master/items", nil); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("kind kosong: status = %d, mau 422", rec.Code)
	}
	if rec := h.req(t, http.MethodGet, "/api/admin/master/items?kind=gadget", nil); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("kind tak dikenal: status = %d, mau 422", rec.Code)
	}
}

// TestAdminCreateItem_Success_FR059 adds a baseline item and checks the 201 body
// carries the contract fields with active true.
func TestAdminCreateItem_Success_FR059(t *testing.T) {
	h := newAdminHarness(t, "admin_create_item")

	rec := h.req(t, http.MethodPost, "/api/admin/master/items", map[string]any{"kind": "machine", "name": "Mesin Obras"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, mau 201; body %s", rec.Code, rec.Body.String())
	}
	var out catalogItem
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("urai body: %v", err)
	}
	if out.ItemID == "" || out.Kind != "machine" || out.Name != "Mesin Obras" || !out.Active {
		t.Fatalf("item = %+v, mau mesin aktif Mesin Obras", out)
	}
}

// TestAdminCreateItem_RejectsRole_FR059 confirms create is admin-gated (403 for a
// buyer).
func TestAdminCreateItem_RejectsRole_FR059(t *testing.T) {
	h := newAdminHarness(t, "admin_create_role")
	h.asBuyer()
	rec := h.req(t, http.MethodPost, "/api/admin/master/items", map[string]any{"kind": "product", "name": "Celana Cargo"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("buyer: status = %d, mau 403", rec.Code)
	}
}

// TestAdminCreateItem_RejectsInvalidInput_FR059 rejects a bad kind, a too-short
// name, and a duplicate name, each a 422.
func TestAdminCreateItem_RejectsInvalidInput_FR059(t *testing.T) {
	h := newAdminHarness(t, "admin_create_invalid")

	if rec := h.req(t, http.MethodPost, "/api/admin/master/items", map[string]any{"kind": "gadget", "name": "Sepatu"}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("kind tak dikenal: status = %d, mau 422", rec.Code)
	}
	if rec := h.req(t, http.MethodPost, "/api/admin/master/items", map[string]any{"kind": "product", "name": "x"}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("nama pendek: status = %d, mau 422", rec.Code)
	}
	if rec := h.req(t, http.MethodPost, "/api/admin/master/items", map[string]any{"kind": "product", "name": "Jaket Denim"}); rec.Code != http.StatusCreated {
		t.Fatalf("seed duplikat: status = %d, mau 201", rec.Code)
	}
	if rec := h.req(t, http.MethodPost, "/api/admin/master/items", map[string]any{"kind": "product", "name": "Jaket Denim"}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("nama duplikat: status = %d, mau 422", rec.Code)
	}
}

// TestAdminUpdateItem_RenamesItem_FR059 renames an existing item through the PATCH
// route and checks the returned body reflects the new name.
func TestAdminUpdateItem_RenamesItem_FR059(t *testing.T) {
	h := newAdminHarness(t, "admin_update_rename")
	itemID := seedCatalogItem(t, h.pool, sqlcgen.ItemTypeProduct, "Kaos Polos")

	rec := h.req(t, http.MethodPatch, "/api/admin/master/items/"+uuidString(itemID), map[string]any{"name": "Kaos Combed"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var out catalogItem
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("urai body: %v", err)
	}
	if out.Name != "Kaos Combed" || !out.Active {
		t.Fatalf("item = %+v, mau Kaos Combed tetap aktif", out)
	}
}

// TestAdminUpdateItem_RejectsRole_FR059 confirms the PATCH route is admin-gated.
func TestAdminUpdateItem_RejectsRole_FR059(t *testing.T) {
	h := newAdminHarness(t, "admin_update_role")
	itemID := seedCatalogItem(t, h.pool, sqlcgen.ItemTypeProduct, "Kaos Raglan")
	h.asBuyer()
	rec := h.req(t, http.MethodPatch, "/api/admin/master/items/"+uuidString(itemID), map[string]any{"active": false})
	if rec.Code != http.StatusForbidden {
		t.Errorf("buyer: status = %d, mau 403", rec.Code)
	}
}

// TestAdminUpdateItem_RejectsInvalidInput_FR059 covers a malformed id (422), an
// empty patch (422), and an unknown item (404).
func TestAdminUpdateItem_RejectsInvalidInput_FR059(t *testing.T) {
	h := newAdminHarness(t, "admin_update_invalid")
	itemID := seedCatalogItem(t, h.pool, sqlcgen.ItemTypeProduct, "Kaos Henley")

	if rec := h.req(t, http.MethodPatch, "/api/admin/master/items/bukan-uuid", map[string]any{"active": false}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("id tidak sah: status = %d, mau 422", rec.Code)
	}
	if rec := h.req(t, http.MethodPatch, "/api/admin/master/items/"+uuidString(itemID), map[string]any{}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("patch kosong: status = %d, mau 422", rec.Code)
	}
	missing := "00000000-0000-0000-0000-000000000000"
	if rec := h.req(t, http.MethodPatch, "/api/admin/master/items/"+missing, map[string]any{"active": false}); rec.Code != http.StatusNotFound {
		t.Errorf("item tak ada: status = %d, mau 404", rec.Code)
	}
}

// TestAdminDeactivateItem_KeepsExistingListingsDiscoverable_FR060 is the critical
// FR-060 gate. Deactivating a catalog item must not touch listings already using
// it, and those listings must stay discoverable through search. It seeds a
// published listing on a product item, deactivates that item through the admin
// PATCH route, and then confirms the same listing is still returned by
// SearchCandidates with a matching score. Search proves the reverse side that a
// new-form pick test cannot: an existing listing survives the deactivation.
func TestAdminDeactivateItem_KeepsExistingListingsDiscoverable_FR060(t *testing.T) {
	h := newAdminHarness(t, "admin_deactivate_discoverable")
	ctx := context.Background()

	productID := seedCatalogItem(t, h.pool, sqlcgen.ItemTypeProduct, "Kemeja Oxford")
	machineID := seedCatalogItem(t, h.pool, sqlcgen.ItemTypeMachine, "Mesin Jahit Lurus")
	listingID := seedDiscoverableListing(t, h.pool, productID, machineID)

	// Deactivate the product item the listing uses, through the real admin route.
	rec := h.req(t, http.MethodPatch, "/api/admin/master/items/"+uuidString(productID), map[string]any{"active": false})
	if rec.Code != http.StatusOK {
		t.Fatalf("nonaktifkan item: status = %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var updated catalogItem
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("urai body: %v", err)
	}
	if updated.Active {
		t.Fatalf("item mau nonaktif setelah PATCH")
	}

	// The listing row and its links must be untouched: still published, still
	// pointing at the now-inactive item.
	var published bool
	if err := h.pool.QueryRow(ctx,
		`SELECT published FROM capacity_listing WHERE id = $1`, listingID).Scan(&published); err != nil {
		t.Fatalf("baca listing: %v", err)
	}
	if !published {
		t.Fatalf("listing mau tetap published setelah item dinonaktifkan")
	}

	// And it must still be discoverable: SearchCandidates ranks it despite the
	// item being inactive. First-page keyset sentinels admit the whole set.
	weekNow := platform.WeekStart(baseTime)
	rows, err := sqlcgen.New(h.pool).SearchCandidates(ctx, sqlcgen.SearchCandidatesParams{
		CursorScore:     5,
		CursorRemaining: 1 << 62,
		CursorNegLead:   1,
		CursorName:      "￿",
		CursorListing:   maxTestUUID(),
		PageSize:        20,
		SearchDate:      pgtype.Date{Time: weekNow, Valid: true},
		DeadlineWeek:    pgtype.Date{Time: weekNow.AddDate(0, 0, 7*4), Valid: true},
		Quantity:        50,
		ProductItem:     productID,
		SearcherProfile: h.buyerProf,
		CityCode:        pgtype.Text{String: "3273", Valid: true},
	})
	if err != nil {
		t.Fatalf("SearchCandidates: %v", err)
	}
	found := false
	for _, r := range rows {
		if r.ListingID == listingID {
			found = true
			if r.Score < 1 {
				t.Errorf("skor kandidat = %d, mau >= 1 (produk cocok)", r.Score)
			}
		}
	}
	if !found {
		t.Fatalf("listing dengan item nonaktif hilang dari pencarian, mau tetap ditemukan (FR-060)")
	}
}

// TestAdminListProposals_Success_FR061 lists a pending proposal for the admin
// review queue, with the proposer's business name attached.
func TestAdminListProposals_Success_FR061(t *testing.T) {
	h := newAdminHarness(t, "admin_list_proposals")
	ctx := context.Background()
	if _, err := h.svc.createProposal(ctx, h.buyerID, proposalInput{Kind: "product", ProposedName: "Rompi Rajut"}); err != nil {
		t.Fatalf("createProposal: %v", err)
	}

	rec := h.req(t, http.MethodGet, "/api/admin/proposals", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var out proposalQueueResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("urai body: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].ProposedName != "Rompi Rajut" || out.Items[0].ProposerName != "Butik Pemesan" {
		t.Fatalf("antrean = %+v, mau satu usulan Rompi Rajut dari Butik Pemesan", out.Items)
	}
}

// TestAdminListProposals_RejectsRole_FR061 confirms the queue is admin-gated.
func TestAdminListProposals_RejectsRole_FR061(t *testing.T) {
	h := newAdminHarness(t, "admin_proposals_role")
	h.asBuyer()
	if rec := h.req(t, http.MethodGet, "/api/admin/proposals", nil); rec.Code != http.StatusForbidden {
		t.Errorf("buyer: status = %d, mau 403", rec.Code)
	}
}

// TestAdminDecideProposal_ApprovesAndNotifies_FR061 approves a pending proposal
// through the route and asserts the proposer received a decision notification.
func TestAdminDecideProposal_ApprovesAndNotifies_FR061(t *testing.T) {
	h := newAdminHarness(t, "admin_decide_approve")
	ctx := context.Background()
	created, err := h.svc.createProposal(ctx, h.buyerID, proposalInput{Kind: "product", ProposedName: "Blazer Formal"})
	if err != nil {
		t.Fatalf("createProposal: %v", err)
	}

	rec := h.req(t, http.MethodPost, "/api/admin/proposals/"+created.ProposalID+"/decision", map[string]any{"decision": "approved"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var out proposalView
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("urai body: %v", err)
	}
	if out.Status != "approved" {
		t.Fatalf("status = %q, mau approved", out.Status)
	}

	var count int
	if err := h.pool.QueryRow(ctx,
		`SELECT count(*) FROM notification WHERE account_id = $1 AND event = $2`,
		h.buyerID, sqlcgen.EventTypeItemProposalDecision).Scan(&count); err != nil {
		t.Fatalf("hitung notifikasi: %v", err)
	}
	if count != 1 {
		t.Errorf("notifikasi proposer = %d, mau 1", count)
	}
}

// TestAdminDecideProposal_RejectsRole_FR061 confirms the decision route is
// admin-gated.
func TestAdminDecideProposal_RejectsRole_FR061(t *testing.T) {
	h := newAdminHarness(t, "admin_decide_role")
	ctx := context.Background()
	created, err := h.svc.createProposal(ctx, h.buyerID, proposalInput{Kind: "product", ProposedName: "Cardigan Wol"})
	if err != nil {
		t.Fatalf("createProposal: %v", err)
	}
	h.asBuyer()
	rec := h.req(t, http.MethodPost, "/api/admin/proposals/"+created.ProposalID+"/decision", map[string]any{"decision": "approved"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("buyer: status = %d, mau 403", rec.Code)
	}
}

// TestAdminDecideProposal_RejectsInvalidInput_FR061 covers a malformed id (422),
// an unknown decision value (422), and a rejection with no reason (422, the
// reason is required before the constraint fires).
func TestAdminDecideProposal_RejectsInvalidInput_FR061(t *testing.T) {
	h := newAdminHarness(t, "admin_decide_invalid")
	ctx := context.Background()
	created, err := h.svc.createProposal(ctx, h.buyerID, proposalInput{Kind: "product", ProposedName: "Sweater Turtleneck"})
	if err != nil {
		t.Fatalf("createProposal: %v", err)
	}

	if rec := h.req(t, http.MethodPost, "/api/admin/proposals/bukan-uuid/decision", map[string]any{"decision": "approved"}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("id tidak sah: status = %d, mau 422", rec.Code)
	}
	if rec := h.req(t, http.MethodPost, "/api/admin/proposals/"+created.ProposalID+"/decision", map[string]any{"decision": "maybe"}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("keputusan tak dikenal: status = %d, mau 422", rec.Code)
	}
	if rec := h.req(t, http.MethodPost, "/api/admin/proposals/"+created.ProposalID+"/decision", map[string]any{"decision": "rejected"}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("tolak tanpa alasan: status = %d, mau 422", rec.Code)
	}
}

// seedDiscoverableListing builds a published listing owned by a fresh
// subcontractor that references productID and machineID, materializing capacity
// weeks from the base week forward so SearchCandidates ranks it. It mirrors the
// search package's seedListing shape. It returns the listing id.
func seedDiscoverableListing(t *testing.T, pool *pgxpool.Pool, productID, machineID pgtype.UUID) pgtype.UUID {
	t.Helper()
	ctx := context.Background()

	acc := seedAccount(t, pool, "subkon-fr060@contoh.test", "628110000050", true)
	var prof pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO business_profile (account_id, business_name, city_code, verified, created_at, updated_at)
		 VALUES ($1, 'Konveksi Discoverable', '3273', false, $2, $2) RETURNING id`,
		acc, baseTime).Scan(&prof); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	weekNow := platform.WeekStart(baseTime)
	horizon := weekNow.AddDate(0, 0, 7*8)

	var listingID pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO capacity_listing (profile_id, weekly_capacity, readiness_lead_days, published, calendar_updated_at, horizon_until, created_at, updated_at)
		 VALUES ($1, 100, 7, true, $2, $3, $2, $2) RETURNING id`,
		prof, baseTime, horizon).Scan(&listingID); err != nil {
		t.Fatalf("seed listing: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO listing_product (listing_id, item_id) VALUES ($1, $2)`,
		listingID, productID); err != nil {
		t.Fatalf("seed listing_product: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO listing_machine (listing_id, item_id, machine_count) VALUES ($1, $2, 1)`,
		listingID, machineID); err != nil {
		t.Fatalf("seed listing_machine: %v", err)
	}
	for w := weekNow; !w.After(horizon); w = w.AddDate(0, 0, 7) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO availability_period (listing_id, week_start, total_capacity, used_capacity, created_at, updated_at)
			 VALUES ($1, $2, 100, 0, $3, $3)`,
			listingID, w, baseTime); err != nil {
			t.Fatalf("seed period: %v", err)
		}
	}
	return listingID
}

// maxTestUUID returns the all-ones UUID, the first-page keyset ceiling for the
// listing tiebreaker so the strict tuple comparison admits every real row.
func maxTestUUID() pgtype.UUID {
	var u pgtype.UUID
	for i := range u.Bytes {
		u.Bytes[i] = 0xFF
	}
	u.Valid = true
	return u
}
