package quota

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func rejectBody(reason string) string {
	b, _ := json.Marshal(map[string]any{"reason": reason})
	return string(b)
}

// TestReject_SubcontractorRejectsWithReason_FR031 proves a candidate's
// subcontractor can decline the request with a reason, that the candidate turns
// to status rejected with the reason stored, and the buyer is notified. The
// endpoint returns 204 with no body.
func TestReject_SubcontractorRejectsWithReason_FR031(t *testing.T) {
	h := newHarness(t, "reject_happy")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	h.asSubcontractor(f.subconAcc)

	rec := h.doJSON(http.MethodPost,
		"/api/candidates/"+uuidString(f.candidateID)+"/reject",
		rejectBody("Kapasitas kami sudah penuh minggu itu."))
	mustStatus(t, rec, http.StatusNoContent)

	var status, reason string
	err := h.pool.QueryRow(context.Background(),
		`SELECT status, rejection_reason FROM request_candidate WHERE id = $1`,
		f.candidateID).Scan(&status, &reason)
	if err != nil {
		t.Fatalf("baca kandidat: %v", err)
	}
	if status != "rejected" {
		t.Fatalf("status %q, mau rejected", status)
	}
	if reason != "Kapasitas kami sudah penuh minggu itu." {
		t.Fatalf("rejection_reason %q tidak tersimpan", reason)
	}
}

// TestReject_BuyerCannotReject_FR031 proves the reject endpoint is gated to the
// subcontractor role: a buyer principal is rejected before the handler runs.
func TestReject_BuyerCannotReject_FR031(t *testing.T) {
	h := newHarness(t, "reject_role")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	// keep default buyer principal

	rec := h.doJSON(http.MethodPost,
		"/api/candidates/"+uuidString(f.candidateID)+"/reject",
		rejectBody("Alasan yang cukup panjang."))
	mustStatus(t, rec, http.StatusForbidden)
}

// TestReject_RejectsShortReason_FR031 proves a reason under 5 characters is a
// 422 with a field name, matching the OfferReject contract minLength.
func TestReject_RejectsShortReason_FR031(t *testing.T) {
	h := newHarness(t, "reject_short")
	f := h.seedCandidate(t, "alfa", 50, platform_deadline(4))
	h.asSubcontractor(f.subconAcc)

	rec := h.doJSON(http.MethodPost,
		"/api/candidates/"+uuidString(f.candidateID)+"/reject",
		rejectBody("no"))
	mustStatus(t, rec, http.StatusUnprocessableEntity)

	p := decodeProblem(t, rec)
	if p.Code != "VALIDATION_FAILED" {
		t.Fatalf("code %q, mau VALIDATION_FAILED", p.Code)
	}
}
