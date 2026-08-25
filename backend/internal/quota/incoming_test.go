package quota

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// incomingResp mirrors IncomingCandidateList: the subcontractor's incoming
// candidates plus the keyset page marker.
type incomingResp struct {
	Items []struct {
		CandidateID  string     `json:"candidate_id"`
		ListingID    string     `json:"listing_id"`
		ProfileID    string     `json:"profile_id"`
		BusinessName string     `json:"business_name"`
		Status       string     `json:"status"`
		LatestOffer  *offerResp `json:"latest_offer"`
	} `json:"items"`
	Pagination struct {
		HasNext    bool    `json:"has_next"`
		NextCursor *string `json:"next_cursor"`
	} `json:"pagination"`
}

func decodeIncoming(t *testing.T, rec *httptest.ResponseRecorder) incomingResp {
	t.Helper()
	var d incomingResp
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatalf("decode incoming %q: %v", rec.Body.String(), err)
	}
	return d
}

// TestIncoming_ListsOwnCandidates_FR030 proves a subcontractor sees the requests
// aimed at their own listing and nobody else's.
func TestIncoming_ListsOwnCandidates_FR030(t *testing.T) {
	h := newHarness(t, "incoming_own")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	h.asSubcontractor(f.subconAcc)

	rec := h.do(http.MethodGet, "/api/quota-requests/incoming")
	mustStatus(t, rec, http.StatusOK)

	d := decodeIncoming(t, rec)
	if len(d.Items) != 1 {
		t.Fatalf("mau 1 kandidat masuk, dapat %d; body %s", len(d.Items), rec.Body.String())
	}
	if d.Items[0].CandidateID != uuidString(f.candidateID) {
		t.Fatalf("candidate_id %q, mau %q", d.Items[0].CandidateID, uuidString(f.candidateID))
	}
	if d.Items[0].Status != "awaiting_reply" {
		t.Fatalf("status %q, mau awaiting_reply", d.Items[0].Status)
	}
}

// TestIncoming_FiltersByStatus_FR031 proves the optional status filter narrows
// the list: after the subcontractor answers one of two candidates, filtering by
// offered returns only the answered one.
func TestIncoming_FiltersByStatus_FR031(t *testing.T) {
	h := newHarness(t, "incoming_filter")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	// A second candidate on the same subcontractor listing stays pending.
	h.seedCandidateFor(t, f.listingID, f.subconAcc, f.subconProf, 30, platform_deadline(4))

	h.asSubcontractor(f.subconAcc)
	rec := h.doJSON(http.MethodPost, "/api/candidates/"+uuidString(f.candidateID)+"/offers", offerBody(3_000_000, 7, ""))
	mustStatus(t, rec, http.StatusCreated)

	rec = h.do(http.MethodGet, "/api/quota-requests/incoming?status=offered")
	mustStatus(t, rec, http.StatusOK)

	d := decodeIncoming(t, rec)
	if len(d.Items) != 1 {
		t.Fatalf("mau 1 kandidat offered, dapat %d; body %s", len(d.Items), rec.Body.String())
	}
	if d.Items[0].CandidateID != uuidString(f.candidateID) {
		t.Fatalf("candidate_id %q, mau %q", d.Items[0].CandidateID, uuidString(f.candidateID))
	}
}

// TestIncoming_RejectsNonSubcontractor_FR030 proves the route is gated to the
// subcontractor role: a buyer principal is rejected before the handler runs.
func TestIncoming_RejectsNonSubcontractor_FR030(t *testing.T) {
	h := newHarness(t, "incoming_role")
	h.asBuyer(h.buyerAcc)

	rec := h.do(http.MethodGet, "/api/quota-requests/incoming")
	mustStatus(t, rec, http.StatusForbidden)
}

// TestIncoming_RejectsInvalidQuery_FR031 proves the query params are validated as
// user input: an unknown status filter and an out-of-range size are each a 422
// with the VALIDATION_FAILED code. FR-031 governs the status filter; no separate
// FR pins query validation, so what is enforced here is the 422 contract response
// for a malformed incoming query.
func TestIncoming_RejectsInvalidQuery_FR031(t *testing.T) {
	h := newHarness(t, "incoming_badquery")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	h.asSubcontractor(f.subconAcc)

	cases := map[string]string{
		"bad status": "/api/quota-requests/incoming?status=ngawur",
		"size zero":  "/api/quota-requests/incoming?size=0",
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
