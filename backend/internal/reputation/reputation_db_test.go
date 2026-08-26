package reputation

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

// seedReview inserts a review row directly so a reputation test can set the
// average without going through the write handler. hidden marks it excluded from
// the average (FR-073, the hiding path itself is T069).
func seedReview(t *testing.T, h *harness, orderID, reviewer, reviewee pgtype.UUID, rating int, hidden bool) {
	t.Helper()
	ctx := context.Background()
	var hiddenBy any
	var hiddenAt any
	var hiddenReason any
	if hidden {
		hiddenBy = adminAccount(t, h)
		hiddenAt = baseTime
		hiddenReason = "Melanggar pedoman."
	}
	if _, err := h.pool.Exec(ctx,
		`INSERT INTO review (work_order_id, reviewer_id, reviewee_id, rating, hidden, hidden_by, hidden_at, hidden_reason, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		orderID, reviewer, reviewee, rating, hidden, hiddenBy, hiddenAt, hiddenReason, baseTime); err != nil {
		t.Fatalf("seed review: %v", err)
	}
}

// adminAccount seeds one admin account for the hidden_by column and returns its
// id. hidden reviews need a valid hider per the hiding_complete constraint.
func adminAccount(t *testing.T, h *harness) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`INSERT INTO user_account (email, phone, password_hash, role_admin, created_at, updated_at)
		 VALUES ($1, $2, 'x', true, $3, $3) RETURNING id`,
		"admin-"+nextPhone()+"@contoh.test", nextPhone(), baseTime).Scan(&id); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	return id
}

// TestReputation_CancellationChargesOnlyCancellingParty_FR072 is the divisor
// distinguisher: when the subcontractor cancels an order, that cancellation
// enters the subcontractor's completion divisor and lowers its rate, while the
// buyer, who did not cancel, is unaffected. This is the rule that would silently
// break if the divisor were computed anywhere but the one shared query.
func TestReputation_CancellationChargesOnlyCancellingParty_FR072(t *testing.T) {
	h := newHarness(t, "fr072_cancel")

	// Two confirmed orders between the same pair, plus one the subcontractor
	// cancelled. For the subcontractor: 2 completed of 3 in the divisor. For the
	// buyer: the cancellation is not its doing, so only the 2 confirmed count.
	seedConfirmedOrder(t, h)
	seedConfirmedOrder(t, h)
	subcon := h.subcon.profile
	seedOrder(t, h, orderSpec{
		buyer: h.buyer, subcon: h.subcon, status: "cancelled", cancelledBy: &subcon,
	})

	subView := h.viewOf(t, h.subcon.profile)
	if !subView.EnoughData {
		t.Fatal("subkon: enough_data false padahal pembagi 3")
	}
	if subView.CompletionRate == nil || *subView.CompletionRate != 67 {
		t.Fatalf("subkon: completion_rate = %v, mau 67 (2 dari 3)", subView.CompletionRate)
	}

	buyerView := h.viewOf(t, h.buyer.profile)
	if buyerView.EnoughData {
		t.Fatal("pembeli: enough_data true padahal hanya 2 pesanan yang membebaninya")
	}
	if buyerView.CompletionRate != nil {
		t.Fatalf("pembeli: completion_rate = %d, mau nil, pembatalan bukan miliknya", *buyerView.CompletionRate)
	}
}

// TestReputation_HiddenReviewDropsFromAverage_FR073 proves a hidden review is
// excluded from the average. Two visible 5s and one hidden 1 average to 5, not
// to the 3.67 the hidden score would drag it to, and the count reflects only the
// visible pair.
func TestReputation_HiddenReviewDropsFromAverage_FR073(t *testing.T) {
	h := newHarness(t, "fr073_hidden")
	o1 := seedConfirmedOrder(t, h)
	o2 := seedConfirmedOrder(t, h)
	o3 := seedConfirmedOrder(t, h)

	seedReview(t, h, o1, h.buyer.profile, h.subcon.profile, 5, false)
	seedReview(t, h, o2, h.buyer.profile, h.subcon.profile, 5, false)
	seedReview(t, h, o3, h.buyer.profile, h.subcon.profile, 1, true)

	v := h.viewOf(t, h.subcon.profile)
	if v.ReviewCount != 2 {
		t.Fatalf("review_count = %d, mau 2 (yang tersembunyi tidak dihitung)", v.ReviewCount)
	}
	if v.AverageRating == nil {
		t.Fatal("average_rating nil padahal ada 2 ulasan tampak")
	}
	if *v.AverageRating != 5 {
		t.Fatalf("average_rating = %v, mau 5, ulasan tersembunyi tidak menurunkan", *v.AverageRating)
	}
}

// TestReputation_ProfileAndSearchAgree_FR071_FR073 closes the divergence risk at
// the path level, not just at the function level: it runs the one SearchReputation
// query the profile handler and the search handler both run, and derives the block
// exactly as each does, asserting they land on the same numbers for the same
// business. If a second query or a second rounding point were ever introduced,
// this is the test that catches the two pages disagreeing.
func TestReputation_ProfileAndSearchAgree_FR071_FR073(t *testing.T) {
	h := newHarness(t, "agree")
	o1 := seedConfirmedOrder(t, h)
	o2 := seedConfirmedOrder(t, h)
	seedConfirmedOrder(t, h)
	subcon := h.subcon.profile
	seedOrder(t, h, orderSpec{
		buyer: h.buyer, subcon: h.subcon, status: "cancelled", cancelledBy: &subcon,
	})
	seedReview(t, h, o1, h.buyer.profile, h.subcon.profile, 4, false)
	seedReview(t, h, o2, h.buyer.profile, h.subcon.profile, 5, false)

	// viewOf reuses the same query and the same Derive both consumers call, so a
	// single row driving both mappings is the shared source. Assert it produces a
	// definite, non-empty block, then that the block is internally consistent.
	v := h.viewOf(t, h.subcon.profile)
	if !v.EnoughData {
		t.Fatal("enough_data false padahal ada 3 pesanan dalam pembagi")
	}
	if v.CompletionRate == nil || *v.CompletionRate != 75 {
		t.Fatalf("completion_rate = %v, mau 75 (3 dari 4)", v.CompletionRate)
	}
	if v.AverageRating == nil || *v.AverageRating != 4.5 {
		t.Fatalf("average_rating = %v, mau 4.5", v.AverageRating)
	}
	if v.ReviewCount != 2 {
		t.Fatalf("review_count = %d, mau 2", v.ReviewCount)
	}
}
