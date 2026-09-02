package reputation

// completionMinOrders is the FR-073 threshold: below three orders in the divisor
// a completion percentage says more about the sample than about the business, so
// it is withheld and the client shows a "belum cukup data" note instead.
const completionMinOrders = 3

// View is the Reputation schema: the read-time figures a profile or a search
// candidate carries (FR-071). Nothing here is stored in a column. completion_rate
// is nil until enough_data is true, matching the contract where only enough_data
// and review_count are required.
type View struct {
	EnoughData     bool     `json:"enough_data"`
	CompletionRate *int     `json:"completion_rate"`
	AverageRating  *float64 `json:"average_rating"`
	ReviewCount    int      `json:"review_count"`
}

// Derive turns the raw counts of one profile into its Reputation block. It is the
// single implementation of the FR-073 threshold and of the percentage rounding,
// so the public profile and the search results cannot disagree about the same
// business. Callers pass scalars they extract from their own query rows; this
// function knows nothing about sqlc.
//
// It does not recompute the divisor. The FR-072 rule (a cancellation enters the
// divisor only for the party that cancelled) lives in SQL, and applying it in two
// places would be the same defect this function exists to prevent: the caller
// passes numbers that are already correct.
//
// Rounding is split deliberately and the split must stay put: average_rating is
// rounded in SQL with round(..., 2) because the average is computed there, while
// completion_rate is rounded here because the percentage is computed here. Moving
// either one to the other side reintroduces two answers for one number.
//
// A divisor at or above the threshold always yields a percentage, including zero:
// a business that cancelled every order has a 0% completion rate, which is not the
// same statement as "not enough data yet".
func Derive(completed, divisor, reviewCount int, avgRating *float64) View {
	v := View{ReviewCount: reviewCount}
	if reviewCount > 0 {
		v.AverageRating = avgRating
	}
	if divisor >= completionMinOrders {
		v.EnoughData = true
		pct := (completed*100 + divisor/2) / divisor
		v.CompletionRate = &pct
	}
	return v
}
