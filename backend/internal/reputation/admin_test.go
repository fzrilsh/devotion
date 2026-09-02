package reputation

import (
	"context"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

// listProfile fetches the subcontractor's public review list and returns its
// items, so a test asserts on exactly what a prospective buyer would see.
func (h *harness) listProfile(t *testing.T) []reviewView {
	t.Helper()
	rec := h.do(http.MethodGet, "/api/profile/"+uuidString(h.subcon.profile)+"/reviews", nil)
	mustStatus(t, rec, http.StatusOK)
	return decodeList(t, rec).Items
}

// mustUUID parses canonical UUID text or fails the test.
func mustUUID(t *testing.T, s string) pgtype.UUID {
	t.Helper()
	u, ok := parseUUID(s)
	if !ok {
		t.Fatalf("uuid tidak sah: %q", s)
	}
	return u
}

// TestHideReview_HidesFromProfileAndSearch_FR050 is the FR-050 special test.
// Hiding one review must remove it from the public profile list AND change the
// average rating, both consistently, because a single NOT hidden filter feeds
// both. It seeds two reviews of the subcontractor with different ratings, checks
// the average and the list before, hides the higher-rated one through the admin
// route, then checks the average moved to the surviving rating and the hidden
// review is gone from the list. The average is read through viewOf, which runs
// the same SearchReputation query the search page and profile use, so this
// proves there is no second filter that could drift.
func TestHideReview_HidesFromProfileAndSearch_FR050(t *testing.T) {
	h := newHarness(t, "hide_review_avg")
	adminID := seedAdmin(t, h.pool, "admin-hide")

	// Two reviews of the subcontractor: rating 5 from the buyer, rating 1 from a
	// second buyer. Average starts at 3.00.
	orderA := seedConfirmedOrder(t, h)
	h.auth.principal = buyerPrincipal(h.buyer.account)
	mustStatus(t, h.postReview(orderA, map[string]any{"rating": 5}), http.StatusCreated)

	other := seedParty(t, h.pool, "pembeli2-hide_review_avg", false)
	orderB := seedOrder(t, h, orderSpec{buyer: other, subcon: h.subcon, status: "confirmed", confirmedAt: &baseTime})
	h.auth.principal = buyerPrincipal(other.account)
	recB := h.postReview(orderB, map[string]any{"rating": 1})
	mustStatus(t, recB, http.StatusCreated)
	hiddenID := decodeReview(t, recB).ReviewID

	// Before hiding: average is 3.00, both reviews visible in the list.
	before := h.viewOf(t, h.subcon.profile)
	if before.AverageRating == nil || *before.AverageRating != 3.0 {
		t.Fatalf("rata-rata sebelum = %v, mau 3.0", before.AverageRating)
	}
	if before.ReviewCount != 2 {
		t.Fatalf("jumlah ulasan sebelum = %d, mau 2", before.ReviewCount)
	}
	if got := h.listProfile(t); len(got) != 2 {
		t.Fatalf("ulasan tampil sebelum = %d, mau 2", len(got))
	}

	// Hide the rating-1 review through the admin route.
	h.auth.principal = adminPrincipal(adminID)
	rec := h.do(http.MethodPost, "/api/admin/reviews/"+hiddenID+"/hide",
		map[string]any{"reason": "Ulasan berisi ujaran kebencian."})
	mustStatus(t, rec, http.StatusOK)
	if v := decodeReview(t, rec); !v.Hidden {
		t.Fatalf("ulasan hasil = %+v, mau hidden true", v)
	}

	// After hiding: average moved to the surviving rating (5.00) and the list
	// drops to one entry, the two derived from the one query and so consistent.
	after := h.viewOf(t, h.subcon.profile)
	if after.AverageRating == nil || *after.AverageRating != 5.0 {
		t.Fatalf("rata-rata sesudah = %v, mau 5.0", after.AverageRating)
	}
	if after.ReviewCount != 1 {
		t.Fatalf("jumlah ulasan sesudah = %d, mau 1", after.ReviewCount)
	}
	if got := h.listProfile(t); len(got) != 1 {
		t.Fatalf("ulasan tampil sesudah = %d, mau 1", len(got))
	}

	// The hidden columns are recorded: admin identity, moment, and reason.
	var by string
	var reason string
	if err := h.pool.QueryRow(context.Background(),
		`SELECT hidden_by::text, hidden_reason FROM review WHERE id = $1`, mustUUID(t, hiddenID)).Scan(&by, &reason); err != nil {
		t.Fatalf("baca kolom hidden: %v", err)
	}
	if by != uuidString(adminID) || reason != "Ulasan berisi ujaran kebencian." {
		t.Fatalf("kolom hidden = (%s, %q), mau admin dan alasan terisi", by, reason)
	}
}

// TestHideReview_RejectsRole_FR050 confirms the hide route is admin-gated: a
// buyer is 403 and an anonymous caller 401.
func TestHideReview_RejectsRole_FR050(t *testing.T) {
	h := newHarness(t, "hide_review_role")
	order := seedConfirmedOrder(t, h)
	h.auth.principal = buyerPrincipal(h.buyer.account)
	reviewID := decodeReview(t, h.postReview(order, map[string]any{"rating": 4})).ReviewID

	path := "/api/admin/reviews/" + reviewID + "/hide"
	body := map[string]any{"reason": "Alasan cukup panjang."}

	// Still a buyer: 403.
	if rec := h.do(http.MethodPost, path, body); rec.Code != http.StatusForbidden {
		t.Errorf("buyer: status = %d, mau 403", rec.Code)
	}
	h.auth.principal = subconPrincipal(h.subcon.account)
	if rec := h.do(http.MethodPost, path, body); rec.Code != http.StatusForbidden {
		t.Errorf("subkon: status = %d, mau 403", rec.Code)
	}
	h.auth.principal = nil
	if rec := h.do(http.MethodPost, path, body); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonim: status = %d, mau 401", rec.Code)
	}
}

// TestHideReview_RejectsInvalidInput_FR050 covers a malformed id (422), a missing
// reason (422, required before the hiding_complete CHECK fires), a too-short
// reason (422), and an unknown review (404).
func TestHideReview_RejectsInvalidInput_FR050(t *testing.T) {
	h := newHarness(t, "hide_review_invalid")
	adminID := seedAdmin(t, h.pool, "admin-invalid")
	h.auth.principal = adminPrincipal(adminID)

	if rec := h.do(http.MethodPost, "/api/admin/reviews/bukan-uuid/hide", map[string]any{"reason": "Alasan cukup."}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("id tidak sah: status = %d, mau 422", rec.Code)
	}

	order := seedConfirmedOrder(t, h)
	h.auth.principal = buyerPrincipal(h.buyer.account)
	reviewID := decodeReview(t, h.postReview(order, map[string]any{"rating": 4})).ReviewID
	h.auth.principal = adminPrincipal(adminID)

	if rec := h.do(http.MethodPost, "/api/admin/reviews/"+reviewID+"/hide", map[string]any{}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("alasan kosong: status = %d, mau 422", rec.Code)
	}
	if rec := h.do(http.MethodPost, "/api/admin/reviews/"+reviewID+"/hide", map[string]any{"reason": "abcd"}); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("alasan pendek: status = %d, mau 422", rec.Code)
	}
	missing := "00000000-0000-0000-0000-000000000000"
	if rec := h.do(http.MethodPost, "/api/admin/reviews/"+missing+"/hide", map[string]any{"reason": "Alasan cukup panjang."}); rec.Code != http.StatusNotFound {
		t.Errorf("ulasan tak ada: status = %d, mau 404", rec.Code)
	}
}
