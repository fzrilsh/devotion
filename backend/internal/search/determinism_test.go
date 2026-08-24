package search

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// TestSearch_OrderRepeatableAndStableAcrossInsertion_SC013_FR025 proves the two
// halves of SC-013: the same query run twice yields the identical order, and a
// listing inserted mid-walk never makes an already-returned candidate reappear
// or drop out. The listing_id final tiebreaker (FR-025) is what makes the order
// total, so a repeat cannot swap two equal-scored rows.
func TestSearch_OrderRepeatableAndStableAcrossInsertion_SC013_FR025(t *testing.T) {
	h := newHarness(t, "search_stable")
	const total = 6
	for i := 0; i < total; i++ {
		// Identical weekly capacity and lead so every candidate ties on score and
		// remaining, forcing the name then listing_id tiebreakers to decide order.
		seedListing(t, h, "sub-"+string(rune('a'+i)), 100, 7, 8)
	}

	// Same query twice: byte-identical order.
	first := decodeResult(t, h.do(http.MethodGet, h.searchPath(10, 4)))
	second := decodeResult(t, h.do(http.MethodGet, h.searchPath(10, 4)))
	if a, b := orderIDs(first), orderIDs(second); !equalStrings(a, b) {
		t.Fatalf("urutan berubah pada pengulangan:\n%v\n%v", a, b)
	}

	// Walk the first page, insert a new listing whose name sorts before the ones
	// already returned, then keep walking. The inserted row must not appear on a
	// page we already passed, and no earlier candidate may repeat or vanish.
	page1 := decodeResult(t, h.do(http.MethodGet, h.searchPath(10, 4)+"&size=2"))
	if len(page1.Items) != 2 || page1.Pagination.NextCursor == nil {
		t.Fatalf("halaman pertama tak lengkap: %+v", page1)
	}
	seedListing(t, h, "sub-000-inserted", 100, 7, 8) // sorts first by name

	seen := map[string]bool{}
	for _, it := range page1.Items {
		seen[it.ListingID] = true
	}
	cursor := *page1.Pagination.NextCursor
	for {
		res := decodeResult(t, h.do(http.MethodGet, cursorPath(h, cursor)))
		for _, it := range res.Items {
			if seen[it.ListingID] {
				t.Fatalf("listing %s muncul dua kali antar halaman", it.ListingID)
			}
			seen[it.ListingID] = true
		}
		if !res.Pagination.HasNext {
			break
		}
		cursor = *res.Pagination.NextCursor
	}
	// The original six are each seen exactly once; the mid-walk insertion sorts
	// before the cursor position, so keyset correctly never surfaces it here.
	if len(seen) < total {
		t.Fatalf("mau minimal %d kandidat unik, dapat %d", total, len(seen))
	}
}

// TestSearch_ScoreIgnoresRatingVerificationCalendar_FR024 mutates the three
// signals the design forbids from touching the score, rating, verification, and
// calendar recency, and asserts the score and the whole order are unchanged.
func TestSearch_ScoreIgnoresRatingVerificationCalendar_FR024(t *testing.T) {
	h := newHarness(t, "search_score_indep")
	a := seedListing(t, h, "sub-a", 100, 7, 8)
	b := seedListing(t, h, "sub-b", 100, 7, 8)

	before := decodeResult(t, h.do(http.MethodGet, h.searchPath(50, 4)))
	if len(before.Items) != 2 {
		t.Fatalf("mau 2 kandidat, dapat %d", len(before.Items))
	}
	beforeScoreA := itemByListing(t, before, a).Score

	ctx := context.Background()
	// Verify listing a's profile, backdate its calendar to look stale, and hand
	// it a one-star review; none of these are inputs to the score.
	profA := profileOfListing(t, h, a)
	if _, err := h.pool.Exec(ctx,
		`UPDATE business_profile SET verified = true WHERE id = $1`, profA); err != nil {
		t.Fatalf("set verified: %v", err)
	}
	if _, err := h.pool.Exec(ctx,
		`UPDATE capacity_listing SET calendar_updated_at = $2 WHERE id = $1`,
		a, baseTime.AddDate(-1, 0, 0)); err != nil {
		t.Fatalf("backdate calendar: %v", err)
	}
	seedReview(t, h, a, profA, 1)

	after := decodeResult(t, h.do(http.MethodGet, h.searchPath(50, 4)))
	if s := itemByListing(t, after, a).Score; s != beforeScoreA {
		t.Fatalf("skor kandidat a berubah dari %d jadi %d setelah rating/verifikasi/kalender diubah", beforeScoreA, s)
	}
	if x, y := orderIDs(before), orderIDs(after); !equalStrings(x, y) {
		t.Fatalf("urutan berubah setelah sinyal non-skor diubah:\n%v\n%v", x, y)
	}
	_ = b
}

// TestSearch_CapacityMeetsAtEightWeeksNotFour_SC019 is the headline capacity
// range case: 3.000 pieces at 500/week, lead 0. With a deadline eight weeks out
// the readiness..deadline range holds enough weeks to pass criterion (d); with a
// four week deadline it does not.
func TestSearch_CapacityMeetsAtEightWeeksNotFour_SC019(t *testing.T) {
	h := newHarness(t, "search_sc019")
	listingID := seedListing(t, h, "pabrik", 500, 0, 12)

	// Eight-week deadline: weeks 0..8 inclusive at 500 = 4500 >= 3000.
	pass := decodeResult(t, h.do(http.MethodGet, h.searchPath(3000, 8)))
	cand := itemByListing(t, pass, listingID)
	if cand.TotalCapacityUntilDeadline < 3000 {
		t.Fatalf("deadline 8 minggu: total %d, mau >= 3000", cand.TotalCapacityUntilDeadline)
	}
	if !criterionByName(t, cand, "Kapasitas mencukupi").Met {
		t.Fatalf("deadline 8 minggu: kriteria kapasitas mau terpenuhi")
	}

	// Four-week deadline: weeks 0..4 inclusive at 500 = 2500 < 3000.
	fail := decodeResult(t, h.do(http.MethodGet, h.searchPath(3000, 4)))
	cand = itemByListing(t, fail, listingID)
	if cand.TotalCapacityUntilDeadline >= 3000 {
		t.Fatalf("deadline 4 minggu: total %d, mau < 3000", cand.TotalCapacityUntilDeadline)
	}
	if criterionByName(t, cand, "Kapasitas mencukupi").Met {
		t.Fatalf("deadline 4 minggu: kriteria kapasitas mau tidak terpenuhi")
	}
}

// TestSearch_ReadinessLeadDropsEarlyWeeks_SC020 confirms a 14-day readiness lead
// excludes the first two weeks from the sum, so on the same deadline its total
// is strictly below an otherwise identical zero-lead candidate.
func TestSearch_ReadinessLeadDropsEarlyWeeks_SC020(t *testing.T) {
	h := newHarness(t, "search_sc020")
	zero := seedListing(t, h, "jeda-nol", 100, 0, 12)
	// 14 days from a Monday base lands on the Monday two weeks out, so the
	// readiness week is week 2 and weeks 0 and 1 are excluded.
	late := seedListing(t, h, "jeda-14", 100, 14, 12)

	res := decodeResult(t, h.do(http.MethodGet, h.searchPath(100, 8)))
	zeroCap := itemByListing(t, res, zero).TotalCapacityUntilDeadline
	lateCap := itemByListing(t, res, late).TotalCapacityUntilDeadline
	if !(lateCap < zeroCap) {
		t.Fatalf("jeda 14 hari (%d) mau lebih kecil dari jeda 0 hari (%d)", lateCap, zeroCap)
	}
	// weeks 0..8 = 9 weeks vs weeks 2..8 = 7 weeks, both at 100.
	if zeroCap != 9*100 {
		t.Fatalf("jeda 0: total %d, mau 900", zeroCap)
	}
	if lateCap != 7*100 {
		t.Fatalf("jeda 14: total %d, mau 700", lateCap)
	}
}

// TestSearch_ReadinessBeyondDeadline_ZeroCapacity_SC020 covers the boundary the
// same design note flags: when the readiness week falls past the deadline week
// the range is empty, so capacity is zero and criterion (d) fails.
func TestSearch_ReadinessBeyondDeadline_ZeroCapacity_SC020(t *testing.T) {
	h := newHarness(t, "search_beyond")
	// 28-day lead puts the readiness week at week 4, past a two-week deadline.
	listingID := seedListing(t, h, "jeda-jauh", 100, 28, 12)

	res := decodeResult(t, h.do(http.MethodGet, h.searchPath(50, 2)))
	cand := itemByListing(t, res, listingID)
	if cand.TotalCapacityUntilDeadline != 0 {
		t.Fatalf("kapasitas mau 0 saat minggu kesiapan melampaui deadline, dapat %d", cand.TotalCapacityUntilDeadline)
	}
	if criterionByName(t, cand, "Kapasitas mencukupi").Met {
		t.Fatalf("kriteria (d) mau tidak terpenuhi saat rentang kapasitas kosong")
	}
}

// TestSearch_FullCapacityBeyondHorizon_ExtendsCalendar_SC021 seeds a listing
// whose materialized horizon is shorter than the deadline, then searches with a
// deadline five months out. The uncreated weeks up to the deadline are counted
// at full capacity, so the candidate passes, and the passing candidate's real
// periods are materialized up to the deadline afterwards (FR-088).
func TestSearch_FullCapacityBeyondHorizon_ExtendsCalendar_SC021(t *testing.T) {
	h := newHarness(t, "search_sc021")
	useRealHorizon(t, h)

	const horizonWeeks = 8
	const deadlineWeeks = 22 // roughly five months, past the seeded horizon
	listingID := seedListing(t, h, "horizon-pendek", 200, 0, horizonWeeks)

	res := decodeResult(t, h.do(http.MethodGet, h.searchPath(3000, deadlineWeeks)))
	cand := itemByListing(t, res, listingID)

	// weeks 0..22 inclusive at 200 = 4600 >= 3000, even though only weeks 0..8
	// (1800) are materialized: the range is scored at full capacity to the deadline.
	wantFull := int64((deadlineWeeks + 1) * 200)
	if cand.TotalCapacityUntilDeadline != wantFull {
		t.Fatalf("total %d, mau %d (kapasitas penuh sampai deadline)", cand.TotalCapacityUntilDeadline, wantFull)
	}
	if !criterionByName(t, cand, "Kapasitas mencukupi").Met {
		t.Fatalf("kriteria kapasitas mau terpenuhi di luar horizon awal")
	}

	// The passing candidate's calendar is now extended to the deadline week: the
	// periods that were only counted optimistically really exist after the search.
	// availability_period.week_start is a date, so compare the calendar day rather
	// than the instant (pgx reads dates back in UTC, WeekStart builds them in WIB).
	deadlineWeek := platform.WeekStart(baseTime).AddDate(0, 0, 7*deadlineWeeks)
	var maxWeek pgtype.Date
	if err := h.pool.QueryRow(context.Background(),
		`SELECT max(week_start) FROM availability_period WHERE listing_id = $1`, listingID).Scan(&maxWeek); err != nil {
		t.Fatalf("baca max week: %v", err)
	}
	if !maxWeek.Valid || !sameDate(maxWeek.Time, deadlineWeek) {
		t.Fatalf("periode terjauh %v, mau %v setelah perpanjangan horizon", maxWeek.Time, deadlineWeek)
	}
}

// TestSearch_EmptyMachineFilter_CriterionSatisfiedNotEvaluated_FR023 omits the
// machine filter. Criterion (b) is then counted satisfied for every candidate
// and the response says it was not evaluated rather than passed on merit.
func TestSearch_EmptyMachineFilter_CriterionSatisfiedNotEvaluated_FR023(t *testing.T) {
	h := newHarness(t, "search_no_machine")
	seedListing(t, h, "sub-a", 100, 7, 8)

	// searchPath without the machine_item_id parameter.
	path := "/api/search?product_item_id=" + uuidString(h.productID) +
		"&quantity=50&deadline=" + deadlineParam(4) + "&region_level=city&city_code=3273"
	res := decodeResult(t, h.do(http.MethodGet, path))
	if len(res.Items) != 1 {
		t.Fatalf("mau 1 kandidat, dapat %d", len(res.Items))
	}
	c := res.Items[0]
	machine := criterionByName(t, c, "Mesin cocok")
	if !machine.Met {
		t.Fatalf("kriteria mesin mau terpenuhi saat filter kosong")
	}
	if machine.Detail == nil || *machine.Detail == "" {
		t.Fatalf("kriteria mesin mau menyebut tidak dievaluasi, detail kosong")
	}
	// Product filter was supplied, so its detail stays nil: the two are distinct.
	if p := criterionByName(t, c, "Produk cocok"); p.Detail != nil {
		t.Fatalf("kriteria produk yang dinilai mau tanpa detail, dapat %q", *p.Detail)
	}
}

// seedReview builds the minimal request -> offer -> work_order chain a review
// depends on, then inserts a one-to-five review whose reviewee is the listing's
// subcontractor. It exists only so FR-024 can prove rating never feeds the score.
func seedReview(t *testing.T, h *harness, listingID, subconProfile pgtype.UUID, rating int16) {
	t.Helper()
	ctx := context.Background()
	weekNow := platform.WeekStart(baseTime)
	deadline := weekNow.AddDate(0, 0, 7*4)

	var requestID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO quota_request (buyer_id, product_item_id, quantity, material, deadline, reply_due_at, created_at)
		 VALUES ($1, $2, 100, 'katun', $3, $4, $5) RETURNING id`,
		h.buyerProf, h.productID, deadline, baseTime.Add(48*time.Hour), baseTime).Scan(&requestID); err != nil {
		t.Fatalf("seed quota_request: %v", err)
	}

	var candidateID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO request_candidate (request_id, listing_id, subcontractor_id, status, updated_at)
		 VALUES ($1, $2, $3, 'agreed', $4) RETURNING id`,
		requestID, listingID, subconProfile, baseTime).Scan(&candidateID); err != nil {
		t.Fatalf("seed request_candidate: %v", err)
	}

	var offerID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO offer (candidate_id, sequence, proposed_by, total_price, readiness_lead_days, created_at)
		 VALUES ($1, 1, 'subcontractor', 100000, 7, $2) RETURNING id`,
		candidateID, baseTime).Scan(&offerID); err != nil {
		t.Fatalf("seed offer: %v", err)
	}

	var orderID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO work_order (candidate_id, offer_id, buyer_id, subcontractor_id, quantity, total_price,
		     deadline, readiness_week_start, status, created_at)
		 VALUES ($1, $2, $3, $4, 100, 100000, $5, $6, 'confirmed', $7) RETURNING id`,
		candidateID, offerID, h.buyerProf, subconProfile, deadline, weekNow, baseTime).Scan(&orderID); err != nil {
		t.Fatalf("seed work_order: %v", err)
	}

	if _, err := h.pool.Exec(ctx,
		`INSERT INTO review (work_order_id, reviewer_id, reviewee_id, rating, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		orderID, h.buyerProf, subconProfile, rating, baseTime); err != nil {
		t.Fatalf("seed review: %v", err)
	}
}

// equalStrings reports slice equality, for order comparison.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// sameDate reports whether two times fall on the same calendar day. week_start
// is a date column, so pgx reads it back at UTC midnight while WeekStart builds
// the Monday in Asia/Jakarta; comparing the instant would spuriously differ by
// the WIB offset even though both name the same Monday.
func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// cursorPath rebuilds the city-scoped search path carrying an opaque cursor.
func cursorPath(h *harness, cursor string) string {
	return h.searchPath(10, 4) + "&size=2&cursor=" + cursor
}
