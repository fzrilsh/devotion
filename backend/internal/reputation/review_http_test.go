package reputation

import (
	"net/http"
	"testing"
)

// TestCreateReview_ConfirmedOrderAcceptsRating_FR047 is the happy path: on an
// order both parties have confirmed received, the buyer may leave a 1..5 rating
// with optional text, and it comes back not anonymous with the author's name.
func TestCreateReview_ConfirmedOrderAcceptsRating_FR047(t *testing.T) {
	h := newHarness(t, "review_happy")
	orderID := seedConfirmedOrder(t, h)

	text := "Hasil rapi, pengiriman tepat waktu."
	rec := h.postReview(orderID, reviewBody{Rating: 5, Text: &text})
	mustStatus(t, rec, http.StatusCreated)

	v := decodeReview(t, rec)
	if v.Rating != 5 {
		t.Fatalf("rating = %d, mau 5", v.Rating)
	}
	if v.Text == nil || *v.Text != text {
		t.Fatalf("text = %v, mau %q", v.Text, text)
	}
	if v.AuthorProfileID != uuidString(h.buyer.profile) {
		t.Fatalf("author = %s, mau pembeli %s", v.AuthorProfileID, uuidString(h.buyer.profile))
	}
	if v.TargetProfileID != uuidString(h.subcon.profile) {
		t.Fatalf("target = %s, mau subkon %s", v.TargetProfileID, uuidString(h.subcon.profile))
	}
	if v.AuthorBusinessName == "" {
		t.Fatal("author_business_name kosong, ulasan tidak boleh anonim")
	}
}

// TestCreateReview_RejectsUnauthenticated_FR047 proves the write route is gated:
// with no session the router refuses before the handler runs.
func TestCreateReview_RejectsUnauthenticated_FR047(t *testing.T) {
	h := newHarness(t, "review_noauth")
	orderID := seedConfirmedOrder(t, h)
	h.auth.principal = nil

	rec := h.postReview(orderID, reviewBody{Rating: 4})
	mustStatus(t, rec, http.StatusUnauthorized)
	if p := decodeProblem(t, rec); p.Code != "NOT_AUTHENTICATED" {
		t.Fatalf("code = %s, mau NOT_AUTHENTICATED", p.Code)
	}
}

// TestCreateReview_RejectsRatingOutOfRange_FR047 covers invalid input: a rating
// outside 1..5 is a 422 naming the rule, not a stored zero.
func TestCreateReview_RejectsRatingOutOfRange_FR047(t *testing.T) {
	h := newHarness(t, "review_badrating")
	orderID := seedConfirmedOrder(t, h)

	rec := h.postReview(orderID, reviewBody{Rating: 6})
	mustStatus(t, rec, http.StatusUnprocessableEntity)
	if p := decodeProblem(t, rec); p.Code != "VALIDATION_FAILED" {
		t.Fatalf("code = %s, mau VALIDATION_FAILED", p.Code)
	}
}

// TestCreateReview_RejectsUnconfirmedOrder_FR050 is a "Selesai bila" case: a
// review on an order that is not yet confirmed received is refused. The status
// precondition cannot be a CHECK (it reads other tables) so it lives in the
// application, and this proves it holds.
func TestCreateReview_RejectsUnconfirmedOrder_FR050(t *testing.T) {
	h := newHarness(t, "review_unconfirmed")
	orderID := seedOrder(t, h, orderSpec{
		buyer: h.buyer, subcon: h.subcon, status: "production",
	})

	rec := h.postReview(orderID, reviewBody{Rating: 5})
	mustStatus(t, rec, http.StatusConflict)
	if p := decodeProblem(t, rec); p.Code != "WORK_ORDER_NOT_COMPLETED" {
		t.Fatalf("code = %s, mau WORK_ORDER_NOT_COMPLETED", p.Code)
	}
}

// TestCreateReview_RejectsNonParty_FR050 is the other "Selesai bila" case: a
// business that never transacted on this order cannot review it. GetWorkOrderForView
// filters to the two parties, so a stranger reads as not found rather than
// leaking that the order exists.
func TestCreateReview_RejectsNonParty_FR050(t *testing.T) {
	h := newHarness(t, "review_nonparty")
	orderID := seedConfirmedOrder(t, h)

	stranger := seedParty(t, h.pool, "orang-luar", false)
	h.auth.principal = buyerPrincipal(stranger.account)

	rec := h.postReview(orderID, reviewBody{Rating: 5})
	mustStatus(t, rec, http.StatusNotFound)
	if p := decodeProblem(t, rec); p.Code != "NOT_FOUND" {
		t.Fatalf("code = %s, mau NOT_FOUND", p.Code)
	}
}

// TestCreateReview_RejectsSecondReviewBySameParty_FR050 proves the one-per-party
// rule: after the buyer reviews an order, a second attempt by the same party is
// refused by the unique constraint, surfaced as 409.
func TestCreateReview_RejectsSecondReviewBySameParty_FR050(t *testing.T) {
	h := newHarness(t, "review_dup")
	orderID := seedConfirmedOrder(t, h)

	first := h.postReview(orderID, reviewBody{Rating: 5})
	mustStatus(t, first, http.StatusCreated)

	second := h.postReview(orderID, reviewBody{Rating: 3})
	mustStatus(t, second, http.StatusConflict)
	if p := decodeProblem(t, second); p.Code != "REVIEW_ALREADY_SUBMITTED" {
		t.Fatalf("code = %s, mau REVIEW_ALREADY_SUBMITTED", p.Code)
	}
}

// TestCreateReview_AllowsBothPartiesOnSameOrder_FR049 confirms the constraint is
// per reviewer, not per order: buyer and subcontractor each leave one review of
// the counterparty on the same order.
func TestCreateReview_AllowsBothPartiesOnSameOrder_FR049(t *testing.T) {
	h := newHarness(t, "review_bothparties")
	orderID := seedConfirmedOrder(t, h)

	buyerRec := h.postReview(orderID, reviewBody{Rating: 5})
	mustStatus(t, buyerRec, http.StatusCreated)

	h.auth.principal = subconPrincipal(h.subcon.account)
	subRec := h.postReview(orderID, reviewBody{Rating: 4})
	mustStatus(t, subRec, http.StatusCreated)

	sv := decodeReview(t, subRec)
	if sv.AuthorProfileID != uuidString(h.subcon.profile) {
		t.Fatalf("author = %s, mau subkon", sv.AuthorProfileID)
	}
	if sv.TargetProfileID != uuidString(h.buyer.profile) {
		t.Fatalf("target = %s, mau pembeli", sv.TargetProfileID)
	}
}
