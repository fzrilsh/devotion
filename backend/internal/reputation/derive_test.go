package reputation

import "testing"

// TestDerive_WithholdsRateBelowThreeOrders_FR073 pins the lower boundary: two
// orders in the divisor is not enough for a percentage, so enough_data stays
// false and completion_rate stays nil while review_count is still reported.
func TestDerive_WithholdsRateBelowThreeOrders_FR073(t *testing.T) {
	v := Derive(1, 2, 0, nil)
	if v.EnoughData {
		t.Fatalf("enough_data = true untuk pembagi 2, seharusnya false")
	}
	if v.CompletionRate != nil {
		t.Fatalf("completion_rate = %d, seharusnya nil", *v.CompletionRate)
	}
}

// TestDerive_ReportsRateAtThreeOrders_FR073 pins the upper side of the same
// boundary: the third order turns the percentage on. Two of three completed
// rounds to 67, which is also the assertion that the rounding lives here rather
// than in each caller.
func TestDerive_ReportsRateAtThreeOrders_FR073(t *testing.T) {
	v := Derive(2, 3, 0, nil)
	if !v.EnoughData {
		t.Fatalf("enough_data = false untuk pembagi 3, seharusnya true")
	}
	if v.CompletionRate == nil {
		t.Fatal("completion_rate nil padahal data sudah cukup")
	}
	if *v.CompletionRate != 67 {
		t.Fatalf("completion_rate = %d, seharusnya 67", *v.CompletionRate)
	}
}

// TestDerive_ZeroCompletedIsZeroPercentNotUnknown_FR073 is the case most easily
// got wrong: a business that cancelled all three of its orders has a 0%
// completion rate, which is a different statement from "data belum cukup".
// Returning nil here would hide the worst record behind the same note a brand
// new account shows.
func TestDerive_ZeroCompletedIsZeroPercentNotUnknown_FR073(t *testing.T) {
	v := Derive(0, 3, 0, nil)
	if !v.EnoughData {
		t.Fatal("enough_data = false padahal pembaginya 3")
	}
	if v.CompletionRate == nil {
		t.Fatal("completion_rate nil untuk 0 dari 3, seharusnya 0")
	}
	if *v.CompletionRate != 0 {
		t.Fatalf("completion_rate = %d, seharusnya 0", *v.CompletionRate)
	}
}

// TestDerive_NoReviewsLeavesAverageUnset_FR071 keeps an unreviewed business from
// reading as rated zero. Without any review the average is absent, not a score.
func TestDerive_NoReviewsLeavesAverageUnset_FR071(t *testing.T) {
	zero := 0.0
	v := Derive(3, 3, 0, &zero)
	if v.AverageRating != nil {
		t.Fatalf("average_rating = %v, seharusnya nil tanpa ulasan", *v.AverageRating)
	}
	if v.ReviewCount != 0 {
		t.Fatalf("review_count = %d, seharusnya 0", v.ReviewCount)
	}
}

// TestDerive_ReportsAverageOnceReviewed_FR071 is the counterpart: with at least
// one review the average passes through untouched, since it is already rounded
// in SQL and must not be rounded a second time here.
func TestDerive_ReportsAverageOnceReviewed_FR071(t *testing.T) {
	avg := 4.33
	v := Derive(3, 3, 3, &avg)
	if v.AverageRating == nil {
		t.Fatal("average_rating nil padahal ada 3 ulasan")
	}
	if *v.AverageRating != 4.33 {
		t.Fatalf("average_rating = %v, seharusnya 4.33 apa adanya", *v.AverageRating)
	}
}
