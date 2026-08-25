package quota

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// candidateFixture is one seeded candidate: the request it belongs to, the
// candidate row, and the owning subcontractor's account and profile so a test
// can act as that subcontractor.
type candidateFixture struct {
	requestID   pgtype.UUID
	candidateID pgtype.UUID
	listingID   pgtype.UUID
	subconAcc   pgtype.UUID
	subconProf  pgtype.UUID
}

// offerResp is the Offer response body.
type offerResp struct {
	OfferID           string    `json:"offer_id"`
	Party             string    `json:"party"`
	TotalPrice        int64     `json:"total_price"`
	ReadinessLeadDays int       `json:"readiness_lead_days"`
	Sequence          int       `json:"sequence"`
	Note              *string   `json:"note"`
	CreatedAt         time.Time `json:"created_at"`
}

func decodeOffer(t *testing.T, rec *httptest.ResponseRecorder) offerResp {
	t.Helper()
	var o offerResp
	if err := json.Unmarshal(rec.Body.Bytes(), &o); err != nil {
		t.Fatalf("decode offer %q: %v", rec.Body.String(), err)
	}
	return o
}

// asSubcontractor points the harness principal at the given subcontractor
// account so a request hits the incoming-side endpoints as that candidate.
func (h *harness) asSubcontractor(acc pgtype.UUID) {
	h.auth.principal = &httpx.Principal{
		Roles:   httpx.RoleSubcontractor,
		Account: sqlcgen.UserAccount{ID: acc},
	}
}

// seedCandidate seeds a fresh subcontractor listing and a quota request from the
// buyer to it, returning the candidate fixture. quantity and deadline drive the
// FR-035 range checks; the reply window is baseTime + 72h so a fresh clock is
// still inside it.
func (h *harness) seedCandidate(t *testing.T, name string, quantity int32, deadline time.Time) candidateFixture {
	t.Helper()
	ctx := context.Background()
	listingID, subconProf := seedListing(t, h, name)

	var subconAcc pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`SELECT account_id FROM business_profile WHERE id = $1`, subconProf).Scan(&subconAcc); err != nil {
		t.Fatalf("baca account subkon: %v", err)
	}

	return h.seedCandidateFor(t, listingID, subconAcc, subconProf, quantity, deadline)
}

// seedCandidateFor seeds a fresh quota request from the buyer aimed at an
// existing subcontractor listing, returning the candidate fixture. It lets a
// test place several candidates on the same subcontractor so the incoming list
// and its status filter can be exercised.
func (h *harness) seedCandidateFor(t *testing.T, listingID, subconAcc, subconProf pgtype.UUID, quantity int32, deadline time.Time) candidateFixture {
	t.Helper()
	ctx := context.Background()

	var requestID pgtype.UUID
	err := h.pool.QueryRow(ctx,
		`INSERT INTO quota_request (buyer_id, product_item_id, quantity, material, deadline, reply_due_at, created_at)
		 VALUES ($1, $2, $3, 'Katun combed 30s', $4, $5, $6) RETURNING id`,
		h.buyerProf, h.productID, quantity, pgdate(deadline),
		tstz(baseTime.Add(72*time.Hour)), tstz(baseTime)).Scan(&requestID)
	if err != nil {
		t.Fatalf("seed request: %v", err)
	}

	var candidateID pgtype.UUID
	err = h.pool.QueryRow(ctx,
		`INSERT INTO request_candidate (request_id, listing_id, subcontractor_id, updated_at)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		requestID, listingID, subconProf, tstz(baseTime)).Scan(&candidateID)
	if err != nil {
		t.Fatalf("seed candidate: %v", err)
	}

	return candidateFixture{
		requestID: requestID, candidateID: candidateID, listingID: listingID,
		subconAcc: subconAcc, subconProf: subconProf,
	}
}

func offerBody(price int64, leadDays int, note string) string {
	m := map[string]any{"total_price": price, "readiness_lead_days": leadDays}
	if note != "" {
		m["note"] = note
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// TestOffer_SubcontractorRepliesWithPriceAndLead_FR031 proves a candidate's
// subcontractor can reply with a whole-rupiah price and a readiness lead, that
// the offer comes back as the first entry (sequence 1) proposed by the
// subcontractor, the candidate turns to status offered, and the buyer is
// notified.
func TestOffer_SubcontractorRepliesWithPriceAndLead_FR031(t *testing.T) {
	h := newHarness(t, "offer_happy")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	h.asSubcontractor(f.subconAcc)

	rec := h.doJSON(http.MethodPost,
		"/api/candidates/"+uuidString(f.candidateID)+"/offers",
		offerBody(2_500_000, 7, "Bisa mulai minggu depan."))
	mustStatus(t, rec, http.StatusCreated)

	o := decodeOffer(t, rec)
	if o.OfferID == "" {
		t.Fatalf("offer_id kosong; body %s", rec.Body.String())
	}
	if o.Party != "subcontractor" {
		t.Fatalf("party %q, mau subcontractor", o.Party)
	}
	if o.TotalPrice != 2_500_000 {
		t.Fatalf("total_price %d, mau 2500000", o.TotalPrice)
	}
	if o.ReadinessLeadDays != 7 {
		t.Fatalf("readiness_lead_days %d, mau 7", o.ReadinessLeadDays)
	}

	var seq int
	var proposedBy, status string
	err := h.pool.QueryRow(context.Background(),
		`SELECT o.sequence, o.proposed_by, c.status
		 FROM offer o JOIN request_candidate c ON c.id = o.candidate_id
		 WHERE o.candidate_id = $1`, f.candidateID).Scan(&seq, &proposedBy, &status)
	if err != nil {
		t.Fatalf("baca offer: %v", err)
	}
	if seq != 1 {
		t.Fatalf("sequence %d, mau 1 (penawaran pertama)", seq)
	}
	if proposedBy != "subcontractor" {
		t.Fatalf("proposed_by %q, mau subcontractor", proposedBy)
	}
	if status != "offered" {
		t.Fatalf("status kandidat %q, mau offered", status)
	}
	if len(h.notifier.events) != 1 || h.notifier.events[0] != sqlcgen.EventTypeOfferReceived {
		t.Fatalf("mau 1 notifikasi offer_received, dapat %v", h.notifier.events)
	}
}

// TestOffer_BuyerCannotReply_FR031 proves the offer endpoint is gated to the
// subcontractor role: a buyer principal is rejected before the handler runs.
func TestOffer_BuyerCannotReply_FR031(t *testing.T) {
	h := newHarness(t, "offer_role")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	// harness default principal is the buyer; do not switch to subcontractor.

	rec := h.doJSON(http.MethodPost,
		"/api/candidates/"+uuidString(f.candidateID)+"/offers",
		offerBody(2_500_000, 7, ""))
	mustStatus(t, rec, http.StatusForbidden)
}

// TestOffer_RejectsInvalidPrice_FR031 proves a non-positive price is a 422 with
// a field name, since money is a whole rupiah of at least 1 (Rule 3).
func TestOffer_RejectsInvalidPrice_FR031(t *testing.T) {
	h := newHarness(t, "offer_invalid")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	h.asSubcontractor(f.subconAcc)

	rec := h.doJSON(http.MethodPost,
		"/api/candidates/"+uuidString(f.candidateID)+"/offers",
		offerBody(0, 7, ""))
	mustStatus(t, rec, http.StatusUnprocessableEntity)

	p := decodeProblem(t, rec)
	if p.Code != "VALIDATION_FAILED" {
		t.Fatalf("code %q, mau VALIDATION_FAILED", p.Code)
	}
}

// TestOffer_RejectsQuantityBeyondCapacity_FR035 proves an offer is refused when
// the request quantity exceeds the total remaining capacity across the
// readiness..deadline weeks, and that the rejection states the actual remaining
// number, the until-week, and the requested quantity as machine meta fields.
func TestOffer_RejectsQuantityBeyondCapacity_FR035(t *testing.T) {
	h := newHarness(t, "offer_capacity")
	// weekly_capacity 100 over weeks 1..4 (readiness_lead 7d, deadline week 4)
	// leaves 400; asking for 500 must be rejected.
	f := h.seedCandidate(t, "alfa", 500, platform_deadline(4))
	h.asSubcontractor(f.subconAcc)

	rec := h.doJSON(http.MethodPost,
		"/api/candidates/"+uuidString(f.candidateID)+"/offers",
		offerBody(2_500_000, 7, ""))
	mustStatus(t, rec, http.StatusConflict)

	p := decodeProblem(t, rec)
	if p.Code != "INSUFFICIENT_CAPACITY" {
		t.Fatalf("code %q, mau INSUFFICIENT_CAPACITY", p.Code)
	}

	var meta struct {
		Meta struct {
			QuantityRequested int64  `json:"quantity_requested"`
			RemainingCapacity int64  `json:"remaining_capacity"`
			UntilWeek         string `json:"until_week"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
		t.Fatalf("decode meta %q: %v", rec.Body.String(), err)
	}
	if meta.Meta.QuantityRequested != 500 {
		t.Fatalf("quantity_requested %d, mau 500", meta.Meta.QuantityRequested)
	}
	if meta.Meta.RemainingCapacity != 400 {
		t.Fatalf("remaining_capacity %d, mau 400", meta.Meta.RemainingCapacity)
	}
	if meta.Meta.UntilWeek == "" {
		t.Fatalf("until_week kosong; body %s", rec.Body.String())
	}
}

// TestOffer_RejectsReadinessAfterDeadline_FR090 proves an offer whose readiness
// start week falls after the deadline week is refused with READINESS_AFTER_DEADLINE
// (422): production could not begin before the deadline is already past.
func TestOffer_RejectsReadinessAfterDeadline_FR090(t *testing.T) {
	h := newHarness(t, "offer_late")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(2))
	h.asSubcontractor(f.subconAcc)

	rec := h.doJSON(http.MethodPost,
		"/api/candidates/"+uuidString(f.candidateID)+"/offers",
		offerBody(2_500_000, 30, "")) // 30 days out lands past week 2
	mustStatus(t, rec, http.StatusUnprocessableEntity)

	p := decodeProblem(t, rec)
	if p.Code != "READINESS_AFTER_DEADLINE" {
		t.Fatalf("code %q, mau READINESS_AFTER_DEADLINE", p.Code)
	}
}

// TestOffer_RejectsAfterReplyWindow_FR082 proves an offer past the 72-hour reply
// window is refused with REQUEST_EXPIRED (410), quoting the expiry moment.
func TestOffer_RejectsAfterReplyWindow_FR082(t *testing.T) {
	h := newHarness(t, "offer_expired")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	h.asSubcontractor(f.subconAcc)
	h.clock.Advance(73 * time.Hour) // reply_due_at is baseTime + 72h

	rec := h.doJSON(http.MethodPost,
		"/api/candidates/"+uuidString(f.candidateID)+"/offers",
		offerBody(2_500_000, 7, ""))
	mustStatus(t, rec, http.StatusGone)

	p := decodeProblem(t, rec)
	if p.Code != "REQUEST_EXPIRED" {
		t.Fatalf("code %q, mau REQUEST_EXPIRED", p.Code)
	}
}

// platform_deadline mirrors deadlineParam but returns the time so seedCandidate
// can store it as a date.
func platform_deadline(n int) time.Time {
	d, _ := time.Parse(dateFmt, deadlineParam(n))
	return d
}
