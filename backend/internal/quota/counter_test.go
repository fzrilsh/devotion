package quota

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

func counterBody(price int64, note string) string {
	m := map[string]any{"total_price": price}
	if note != "" {
		m["note"] = note
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// asBuyer points the harness principal back at the seeded buyer account, undoing
// a prior asSubcontractor switch within one test.
func (h *harness) asBuyer(acc pgtype.UUID) {
	h.auth.principal = &httpx.Principal{
		Roles:   httpx.RoleBuyer,
		Account: sqlcgen.UserAccount{ID: acc},
	}
}

// firstOffer creates the opening subcontractor offer on a candidate through the
// public endpoint and returns its offer id, so a counter test starts from a real
// chain rather than a hand-seeded row.
func (h *harness) firstOffer(t *testing.T, f candidateFixture, price int64) string {
	t.Helper()
	h.asSubcontractor(f.subconAcc)
	rec := h.doJSON(http.MethodPost,
		"/api/candidates/"+uuidString(f.candidateID)+"/offers",
		offerBody(price, 7, "Penawaran awal."))
	mustStatus(t, rec, http.StatusCreated)
	return decodeOffer(t, rec).OfferID
}

// TestCounter_BuyerCountersSubcontractorOffer_FR033 proves the buyer can answer a
// subcontractor's offer with a new price, that the counter is a new row (sequence
// 2) proposed by the buyer with the earlier offer preserved, and the
// subcontractor is notified of the counter-offer.
func TestCounter_BuyerCountersSubcontractorOffer_FR033(t *testing.T) {
	h := newHarness(t, "counter_happy")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	offerID := h.firstOffer(t, f, 3_000_000)

	h.asBuyer(h.buyerAcc)
	rec := h.doJSON(http.MethodPost,
		"/api/offers/"+offerID+"/counter",
		counterBody(2_400_000, "Bisa turun sedikit?"))
	mustStatus(t, rec, http.StatusCreated)

	o := decodeOffer(t, rec)
	if o.Party != "buyer" {
		t.Fatalf("party %q, mau buyer", o.Party)
	}
	if o.TotalPrice != 2_400_000 {
		t.Fatalf("total_price %d, mau 2400000", o.TotalPrice)
	}
	if o.ReadinessLeadDays != 7 {
		t.Fatalf("readiness_lead_days %d, mau 7 (diwarisi dari penawaran awal)", o.ReadinessLeadDays)
	}

	var n int
	if err := h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM offer WHERE candidate_id = $1`, f.candidateID).Scan(&n); err != nil {
		t.Fatalf("hitung offer: %v", err)
	}
	if n != 2 {
		t.Fatalf("mau 2 baris offer (riwayat tersimpan), dapat %d", n)
	}

	var seq int
	var proposedBy string
	if err := h.pool.QueryRow(context.Background(),
		`SELECT sequence, proposed_by FROM offer WHERE candidate_id = $1 ORDER BY sequence DESC LIMIT 1`,
		f.candidateID).Scan(&seq, &proposedBy); err != nil {
		t.Fatalf("baca offer terbaru: %v", err)
	}
	if seq != 2 {
		t.Fatalf("sequence %d, mau 2", seq)
	}
	if proposedBy != "buyer" {
		t.Fatalf("proposed_by %q, mau buyer", proposedBy)
	}
	if len(h.notifier.events) != 2 || h.notifier.events[1] != sqlcgen.EventTypeCounterOffer {
		t.Fatalf("mau notifikasi kedua counter_offer, dapat %v", h.notifier.events)
	}
}

// TestCounter_ChainAlternatesParties_FR033 proves a counter can be answered by a
// further counter from the opposite side, extending the chain (sequence 3) while
// every earlier round stays on record.
func TestCounter_ChainAlternatesParties_FR033(t *testing.T) {
	h := newHarness(t, "counter_chain")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	offerID := h.firstOffer(t, f, 3_000_000)

	h.asBuyer(h.buyerAcc)
	rec := h.doJSON(http.MethodPost, "/api/offers/"+offerID+"/counter", counterBody(2_400_000, ""))
	mustStatus(t, rec, http.StatusCreated)
	counterID := decodeOffer(t, rec).OfferID

	h.asSubcontractor(f.subconAcc)
	rec = h.doJSON(http.MethodPost, "/api/offers/"+counterID+"/counter", counterBody(2_700_000, "Ketemu tengah."))
	mustStatus(t, rec, http.StatusCreated)

	o := decodeOffer(t, rec)
	if o.Party != "subcontractor" {
		t.Fatalf("party %q, mau subcontractor", o.Party)
	}

	var n int
	if err := h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM offer WHERE candidate_id = $1`, f.candidateID).Scan(&n); err != nil {
		t.Fatalf("hitung offer: %v", err)
	}
	if n != 3 {
		t.Fatalf("mau 3 baris offer, dapat %d", n)
	}
}

// TestCounter_RejectsSamePartyTwice_FR033 proves a party cannot counter its own
// latest offer: negotiation alternates, so the subcontractor countering right
// after its own opening offer is refused.
func TestCounter_RejectsSamePartyTwice_FR033(t *testing.T) {
	h := newHarness(t, "counter_same")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	offerID := h.firstOffer(t, f, 3_000_000)

	h.asSubcontractor(f.subconAcc)
	rec := h.doJSON(http.MethodPost, "/api/offers/"+offerID+"/counter", counterBody(2_800_000, ""))
	mustStatus(t, rec, http.StatusForbidden)
}

// TestCounter_RejectsOutsider_FR033 proves an account that is neither the buyer
// nor the candidate's subcontractor cannot counter the offer.
func TestCounter_RejectsOutsider_FR033(t *testing.T) {
	h := newHarness(t, "counter_outsider")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	offerID := h.firstOffer(t, f, 3_000_000)

	other := seedAccount(t, h.pool, "lain@contoh.test", nextPhone(), false)
	seedProfile(t, h.pool, other, "Pihak Lain", "3273")
	h.asBuyer(other)

	rec := h.doJSON(http.MethodPost, "/api/offers/"+offerID+"/counter", counterBody(2_400_000, ""))
	mustStatus(t, rec, http.StatusForbidden)
}

// TestCounter_RejectsInvalidPrice_FR033 proves a non-positive price is a 422 with
// a field name (money is a whole rupiah of at least 1, Rule 3).
func TestCounter_RejectsInvalidPrice_FR033(t *testing.T) {
	h := newHarness(t, "counter_invalid")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	offerID := h.firstOffer(t, f, 3_000_000)

	h.asBuyer(h.buyerAcc)
	rec := h.doJSON(http.MethodPost, "/api/offers/"+offerID+"/counter", counterBody(0, ""))
	mustStatus(t, rec, http.StatusUnprocessableEntity)

	p := decodeProblem(t, rec)
	if p.Code != "VALIDATION_FAILED" {
		t.Fatalf("code %q, mau VALIDATION_FAILED", p.Code)
	}
}
