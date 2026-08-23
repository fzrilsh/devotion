package masterdata

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/notification"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// mockAuth is a stand-in Authenticator so the masterdata tests never import the
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

// proposalHarness wires a masterdata Service with a real notification service
// against an isolated schema, seeding one buyer account with a profile. auth
// starts as that buyer; a test that checks a role gate swaps auth.principal.
type proposalHarness struct {
	svc       *Service
	handler   http.Handler
	pool      *pgxpool.Pool
	clock     *platform.TestClock
	auth      *mockAuth
	accountID pgtype.UUID
	profileID pgtype.UUID
}

func newProposalHarness(t *testing.T, name string) *proposalHarness {
	t.Helper()
	pool := testdb.New(t, name)
	clock := platform.NewTestClock(baseTime)
	notif := notification.New(pool, clock, nil, nil, nil)
	svc := New(pool, clock, nil, notif)

	accountID := seedAccount(t, pool, "buyer@contoh.test", "628110000009", false)
	profileID := seedProfile(t, pool, accountID, "Butik Pemesan")

	auth := &mockAuth{principal: &httpx.Principal{
		Roles:   httpx.RoleBuyer,
		Account: sqlcgen.UserAccount{ID: accountID},
	}}

	r := httpx.NewRouter(quietLogger())
	svc.Register(r, auth)

	return &proposalHarness{
		svc:       svc,
		handler:   r.Handler(),
		pool:      pool,
		clock:     clock,
		auth:      auth,
		accountID: accountID,
		profileID: profileID,
	}
}

// baseTime is a fixed Monday (2026-08-24) so timestamps are deterministic.
var baseTime = time.Date(2026, 8, 24, 10, 3, 0, 0, time.UTC)

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

// seedProfile inserts the business_profile born with an account and returns its
// id. It points at a province and city seeded here so the FK holds.
func seedProfile(t *testing.T, pool *pgxpool.Pool, accountID pgtype.UUID, name string) pgtype.UUID {
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
	var id pgtype.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO business_profile (account_id, business_name, city_code, created_at, updated_at)
		 VALUES ($1, $2, '3273', $3, $3) RETURNING id`,
		accountID, name, baseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	return id
}

// postProposal drives one POST /api/master/proposals with the given JSON body
// and returns the recorder.
func (h *proposalHarness) postProposal(t *testing.T, body any) *httptest.ResponseRecorder {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/master/proposals", bytes.NewReader(buf))
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

// TestCreateProposal_Success_FR061 posts a valid proposal as a buyer and checks
// the 201 body carries the contract fields with a pending status.
func TestCreateProposal_Success_FR061(t *testing.T) {
	h := newProposalHarness(t, "proposal_success")

	rec := h.postProposal(t, map[string]any{"kind": "product", "proposed_name": "Jaket Bomber"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, mau 201; body %s", rec.Code, rec.Body.String())
	}

	var out proposalView
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("urai body: %v", err)
	}
	if out.ProposalID == "" {
		t.Error("proposal_id kosong")
	}
	if out.Kind != "product" {
		t.Errorf("kind = %q, mau product", out.Kind)
	}
	if out.ProposedName != "Jaket Bomber" {
		t.Errorf("proposed_name = %q, mau Jaket Bomber", out.ProposedName)
	}
	if out.Status != "pending" {
		t.Errorf("status = %q, mau pending", out.Status)
	}
	if out.Reason != nil {
		t.Errorf("reason = %v, mau null", *out.Reason)
	}
}

// TestCreateProposal_RejectsRole_FR061 confirms an unauthenticated caller is
// 401 and an admin (neither business role) is 403: the proposal path is gated
// to the two business roles.
func TestCreateProposal_RejectsRole_FR061(t *testing.T) {
	h := newProposalHarness(t, "proposal_role")
	body := map[string]any{"kind": "product", "proposed_name": "Kemeja Flanel"}

	h.auth.principal = nil
	if rec := h.postProposal(t, body); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonim: status = %d, mau 401", rec.Code)
	}

	h.auth.principal = &httpx.Principal{Roles: httpx.RoleAdmin, Account: sqlcgen.UserAccount{ID: h.accountID}}
	if rec := h.postProposal(t, body); rec.Code != http.StatusForbidden {
		t.Errorf("admin: status = %d, mau 403", rec.Code)
	}
}

// TestCreateProposal_RejectsInvalidInput_FR061 rejects a too-short name and an
// unknown kind, each a 422.
func TestCreateProposal_RejectsInvalidInput_FR061(t *testing.T) {
	h := newProposalHarness(t, "proposal_invalid")

	if rec := h.postProposal(t, map[string]any{"kind": "product", "proposed_name": "x"}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("nama pendek: status = %d, mau 422", rec.Code)
	}
	if rec := h.postProposal(t, map[string]any{"kind": "gadget", "proposed_name": "Sepatu Kulit"}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("kind tak dikenal: status = %d, mau 422", rec.Code)
	}
}

// TestDecideProposal_NotifiesProposer_FR061 creates a proposal, decides it as an
// admin, and asserts the proposer received an item_proposal_decision
// notification row written inside the decision transaction.
func TestDecideProposal_NotifiesProposer_FR061(t *testing.T) {
	h := newProposalHarness(t, "proposal_decide")
	ctx := context.Background()

	created, err := h.svc.createProposal(ctx, h.accountID, proposalInput{Kind: "product", ProposedName: "Rok Plisket"})
	if err != nil {
		t.Fatalf("createProposal: %v", err)
	}
	proposalID, ok := parseTestUUID(t, created.ProposalID)
	if !ok {
		return
	}

	adminID := seedAccount(t, h.pool, "admin@contoh.test", "628110000010", false)
	itemID := seedCatalogItem(t, h.pool, sqlcgen.ItemTypeProduct, "Rok Plisket")

	view, err := h.svc.DecideProposal(ctx, proposalID, adminID, true, nil, itemID)
	if err != nil {
		t.Fatalf("DecideProposal: %v", err)
	}
	if view.Status != "approved" {
		t.Errorf("status = %q, mau approved", view.Status)
	}

	var count int
	err = h.pool.QueryRow(ctx,
		`SELECT count(*) FROM notification WHERE account_id = $1 AND event = $2`,
		h.accountID, sqlcgen.EventTypeItemProposalDecision).Scan(&count)
	if err != nil {
		t.Fatalf("hitung notifikasi: %v", err)
	}
	if count != 1 {
		t.Errorf("notifikasi proposer = %d, mau 1", count)
	}
}

// seedCatalogItem inserts an active catalog item and returns its id.
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

// parseTestUUID parses canonical UUID text into a pgtype.UUID for a test.
func parseTestUUID(t *testing.T, s string) (pgtype.UUID, bool) {
	t.Helper()
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		t.Fatalf("parse uuid %q: %v", s, err)
		return pgtype.UUID{}, false
	}
	return u, true
}
