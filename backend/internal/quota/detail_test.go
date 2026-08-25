package quota

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// detailResp mirrors QuotaRequestDetail: the request fields plus each candidate
// with its latest offer attached so the buyer compares them side by side.
type detailResp struct {
	RequestID  string `json:"request_id"`
	Quantity   int    `json:"quantity"`
	Material   string `json:"material"`
	Candidates []struct {
		CandidateID  string      `json:"candidate_id"`
		ListingID    string      `json:"listing_id"`
		ProfileID    string      `json:"profile_id"`
		BusinessName string      `json:"business_name"`
		Status       string      `json:"status"`
		Offers       []offerResp `json:"offers"`
		LatestOffer  *offerResp  `json:"latest_offer"`
	} `json:"candidates"`
}

func decodeDetail(t *testing.T, rec *httptest.ResponseRecorder) detailResp {
	t.Helper()
	var d detailResp
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatalf("decode detail %q: %v", rec.Body.String(), err)
	}
	return d
}

// TestDetail_ShowsLatestOfferPerCandidate_FR032 proves the buyer's detail view
// returns every candidate with its most recent offer attached, so the offers are
// visible side by side (FR-032). It seeds one candidate, has the subcontractor
// open an offer and the buyer counter it, then reads the detail and expects the
// counter (the latest round) as latest_offer.
func TestDetail_ShowsLatestOfferPerCandidate_FR032(t *testing.T) {
	h := newHarness(t, "detail_offers")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	offerID := h.firstOffer(t, f, 3_000_000)

	h.asBuyer(h.buyerAcc)
	rec := h.doJSON(http.MethodPost, "/api/offers/"+offerID+"/counter", counterBody(2_400_000, ""))
	mustStatus(t, rec, http.StatusCreated)

	rec = h.do(http.MethodGet, "/api/quota-requests/"+uuidString(f.requestID))
	mustStatus(t, rec, http.StatusOK)

	d := decodeDetail(t, rec)
	if d.RequestID != uuidString(f.requestID) {
		t.Fatalf("request_id %q, mau %q", d.RequestID, uuidString(f.requestID))
	}
	if len(d.Candidates) != 1 {
		t.Fatalf("mau 1 kandidat, dapat %d", len(d.Candidates))
	}
	c := d.Candidates[0]
	if c.LatestOffer == nil {
		t.Fatalf("latest_offer nil; body %s", rec.Body.String())
	}
	if c.LatestOffer.TotalPrice != 2_400_000 {
		t.Fatalf("latest_offer.total_price %d, mau 2400000 (ronde terakhir)", c.LatestOffer.TotalPrice)
	}
	if c.LatestOffer.Party != "buyer" {
		t.Fatalf("latest_offer.party %q, mau buyer", c.LatestOffer.Party)
	}
	if c.Status != "offered" {
		t.Fatalf("status kandidat %q, mau offered", c.Status)
	}
}

// TestDetail_NoOfferYet_FR032 proves a candidate with no reply yet comes back
// with a null latest_offer rather than being dropped from the list.
func TestDetail_NoOfferYet_FR032(t *testing.T) {
	h := newHarness(t, "detail_empty")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))

	rec := h.do(http.MethodGet, "/api/quota-requests/"+uuidString(f.requestID))
	mustStatus(t, rec, http.StatusOK)

	d := decodeDetail(t, rec)
	if len(d.Candidates) != 1 {
		t.Fatalf("mau 1 kandidat, dapat %d", len(d.Candidates))
	}
	if d.Candidates[0].LatestOffer != nil {
		t.Fatalf("latest_offer mau nil sebelum ada balasan, dapat %+v", d.Candidates[0].LatestOffer)
	}
}

// TestDetail_ShowsWholeOfferChainOrdered_FR032_Step37 proves the buyer's detail
// view returns the whole offer chain per candidate ordered by sequence, not just
// the latest round, so the earlier offer stays visible after a counter (manual
// step 3.7: "riwayat penawaran sebelumnya tetap terlihat"). It seeds one
// candidate, the subcontractor opens an offer, the buyer counters, then the
// detail must carry both rounds in offers[] ordered sequence ascending with
// latest_offer pointing at the last (the buyer's counter).
func TestDetail_ShowsWholeOfferChainOrdered_FR032_Step37(t *testing.T) {
	h := newHarness(t, "detail_chain")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	offerID := h.firstOffer(t, f, 3_000_000)

	h.asBuyer(h.buyerAcc)
	rec := h.doJSON(http.MethodPost, "/api/offers/"+offerID+"/counter", counterBody(2_400_000, ""))
	mustStatus(t, rec, http.StatusCreated)

	rec = h.do(http.MethodGet, "/api/quota-requests/"+uuidString(f.requestID))
	mustStatus(t, rec, http.StatusOK)

	d := decodeDetail(t, rec)
	if len(d.Candidates) != 1 {
		t.Fatalf("mau 1 kandidat, dapat %d", len(d.Candidates))
	}
	c := d.Candidates[0]
	if len(c.Offers) != 2 {
		t.Fatalf("mau 2 ronde di offers, dapat %d; body %s", len(c.Offers), rec.Body.String())
	}
	// Both rounds present, ordered by sequence ascending.
	if c.Offers[0].Sequence != 1 || c.Offers[1].Sequence != 2 {
		t.Fatalf("offers tidak terurut sequence: %d lalu %d", c.Offers[0].Sequence, c.Offers[1].Sequence)
	}
	if c.Offers[0].Party != "subcontractor" || c.Offers[0].TotalPrice != 3_000_000 {
		t.Fatalf("ronde 1 salah: party %q harga %d, mau subcontractor 3000000", c.Offers[0].Party, c.Offers[0].TotalPrice)
	}
	if c.Offers[1].Party != "buyer" || c.Offers[1].TotalPrice != 2_400_000 {
		t.Fatalf("ronde 2 salah: party %q harga %d, mau buyer 2400000", c.Offers[1].Party, c.Offers[1].TotalPrice)
	}
	// latest_offer is the last element of the chain, not a separately computed row.
	if c.LatestOffer == nil {
		t.Fatalf("latest_offer nil; body %s", rec.Body.String())
	}
	if c.LatestOffer.Sequence != c.Offers[len(c.Offers)-1].Sequence {
		t.Fatalf("latest_offer.sequence %d, mau elemen terakhir %d", c.LatestOffer.Sequence, c.Offers[len(c.Offers)-1].Sequence)
	}
	if c.LatestOffer.TotalPrice != 2_400_000 {
		t.Fatalf("latest_offer.total_price %d, mau 2400000 (ronde terakhir)", c.LatestOffer.TotalPrice)
	}
}

// TestDetail_RejectsOtherBuyersRequest_FR030 proves a request that is not the
// caller's own is a 404, not a 403: its existence is not leaked to a different
// buyer.
func TestDetail_RejectsOtherBuyersRequest_FR030(t *testing.T) {
	h := newHarness(t, "detail_other")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))

	other := seedAccount(t, h.pool, "pembeli-lain@contoh.test", nextPhone(), false)
	seedProfile(t, h.pool, other, "Pembeli Lain", "3273")
	h.auth.principal = &httpx.Principal{
		Roles:   httpx.RoleBuyer,
		Account: sqlcgen.UserAccount{ID: other},
	}

	rec := h.do(http.MethodGet, "/api/quota-requests/"+uuidString(f.requestID))
	mustStatus(t, rec, http.StatusNotFound)
}

// TestDetail_RejectsNonBuyer_FR030 proves the detail route is gated to the buyer
// role: a subcontractor principal is rejected before the handler runs.
func TestDetail_RejectsNonBuyer_FR030(t *testing.T) {
	h := newHarness(t, "detail_role")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	h.asSubcontractor(f.subconAcc)

	rec := h.do(http.MethodGet, "/api/quota-requests/"+uuidString(f.requestID))
	mustStatus(t, rec, http.StatusForbidden)
}
