package order

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// seedOfferedCandidateLead is seedOfferedCandidate with an explicit readiness
// lead, so a test can push the readiness week past the base week (SC-020).
func seedOfferedCandidateLead(t *testing.T, h *acceptHarness, requestID, listingID, subProfile pgtype.UUID, price int64, leadDays int) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	var candidateID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO request_candidate (request_id, listing_id, subcontractor_id, status, updated_at)
		 VALUES ($1, $2, $3, 'offered', $4) RETURNING id`,
		requestID, listingID, subProfile, acceptBaseTime).Scan(&candidateID); err != nil {
		t.Fatalf("seed candidate: %v", err)
	}
	var offerID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO offer (candidate_id, sequence, proposed_by, total_price, readiness_lead_days, created_at)
		 VALUES ($1, 1, 'subcontractor', $2, $3, $4) RETURNING id`,
		candidateID, price, leadDays, acceptBaseTime).Scan(&offerID); err != nil {
		t.Fatalf("seed offer: %v", err)
	}
	return offerID
}

// usedByWeek returns a listing's used_capacity keyed by the week_start date text,
// so a test can assert the fill landed on the weeks it expects and left the rest
// at zero.
func usedByWeek(t *testing.T, h *acceptHarness, listingID pgtype.UUID) map[string]int32 {
	t.Helper()
	rows, err := h.pool.Query(context.Background(),
		`SELECT week_start, used_capacity FROM availability_period WHERE listing_id = $1`, listingID)
	if err != nil {
		t.Fatalf("baca periode: %v", err)
	}
	defer rows.Close()
	out := map[string]int32{}
	for rows.Next() {
		var wk pgtype.Date
		var used int32
		if err := rows.Scan(&wk, &used); err != nil {
			t.Fatalf("scan periode: %v", err)
		}
		out[wk.Time.Format("2006-01-02")] = used
	}
	return out
}

// weekKey formats a week offset from the base Monday as the map key usedByWeek
// returns.
func weekKey(offsetWeeks int) string {
	return platform.WeekStart(acceptBaseTime).AddDate(0, 0, 7*offsetWeeks).Format("2006-01-02")
}

// TestAccept_FillsEarliestWeeksFirst_FR018_FR078 orders 1,200 pieces against a
// 500/week listing: the fill must take 500 from the first week, 500 from the
// second, 200 from the third, and leave every later week untouched. This is the
// earliest-weeks-first rule (FR-018) with the per-week ceiling (FR-078).
func TestAccept_FillsEarliestWeeksFirst_FR018_FR078(t *testing.T) {
	h := newAcceptHarness(t, "alloc_fill_earliest")
	base := platform.WeekStart(acceptBaseTime)
	deadline := base.AddDate(0, 0, 7*4)

	listingID, subAcc := seedListing(t, h, "alfa", 500, deadline, deadline)
	subProf := subProfileID(t, h, subAcc)

	buyer := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyer@contoh.test", false), "Pembeli")
	req := seedRequest(t, h, buyer, 1200, deadline)
	offer := seedOfferedCandidate(t, h, req, listingID, subProf, 1_000_000)
	acc := buyerAccountOf(t, h, buyer)

	if _, err := h.svc.accept(context.Background(), acc, offer); err != nil {
		t.Fatalf("accept gagal: %v", err)
	}

	used := usedByWeek(t, h, listingID)
	want := map[string]int32{weekKey(0): 500, weekKey(1): 500, weekKey(2): 200, weekKey(3): 0, weekKey(4): 0}
	for wk, w := range want {
		if used[wk] != w {
			t.Fatalf("minggu %s used = %d, mau %d", wk, used[wk], w)
		}
	}
}

// TestAccept_ReadinessLagSkipsEarlyWeeks_FR087_SC020 gives the offer a 14-day
// readiness lead, so production cannot start for two weeks. The allocation must
// begin at the readiness week and leave the first two weeks at zero, even though
// they have free capacity (SC-020, FR-087).
func TestAccept_ReadinessLagSkipsEarlyWeeks_FR087_SC020(t *testing.T) {
	h := newAcceptHarness(t, "alloc_readiness_lag")
	base := platform.WeekStart(acceptBaseTime)
	deadline := base.AddDate(0, 0, 7*3)

	listingID, subAcc := seedListing(t, h, "alfa", 500, deadline, deadline)
	subProf := subProfileID(t, h, subAcc)

	buyer := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyer@contoh.test", false), "Pembeli")
	req := seedRequest(t, h, buyer, 500, deadline)
	// 14 days lands on the Monday two weeks out, so readiness_week_start is week 2.
	offer := seedOfferedCandidateLead(t, h, req, listingID, subProf, 1_000_000, 14)
	acc := buyerAccountOf(t, h, buyer)

	if _, err := h.svc.accept(context.Background(), acc, offer); err != nil {
		t.Fatalf("accept gagal: %v", err)
	}

	used := usedByWeek(t, h, listingID)
	if used[weekKey(0)] != 0 || used[weekKey(1)] != 0 {
		t.Fatalf("dua minggu pertama used = %d/%d, mau 0/0 (kesiapan belum tercapai)", used[weekKey(0)], used[weekKey(1)])
	}
	if used[weekKey(2)] != 500 {
		t.Fatalf("minggu kesiapan used = %d, mau 500", used[weekKey(2)])
	}
}

// TestAccept_SkipsMarkedFullWeek_FR078 marks the first week full before the
// order. The fill must skip it and move the whole quantity to the next week,
// leaving the marked-full week untouched (FR-078).
func TestAccept_SkipsMarkedFullWeek_FR078(t *testing.T) {
	h := newAcceptHarness(t, "alloc_skip_full")
	base := platform.WeekStart(acceptBaseTime)
	deadline := base.AddDate(0, 0, 7*2)

	listingID, subAcc := seedListing(t, h, "alfa", 500, deadline, deadline)
	subProf := subProfileID(t, h, subAcc)

	if _, err := h.pool.Exec(context.Background(),
		`UPDATE availability_period SET marked_full = true WHERE listing_id = $1 AND week_start = $2`,
		listingID, base); err != nil {
		t.Fatalf("tandai penuh: %v", err)
	}

	buyer := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyer@contoh.test", false), "Pembeli")
	req := seedRequest(t, h, buyer, 500, deadline)
	offer := seedOfferedCandidate(t, h, req, listingID, subProf, 1_000_000)
	acc := buyerAccountOf(t, h, buyer)

	if _, err := h.svc.accept(context.Background(), acc, offer); err != nil {
		t.Fatalf("accept gagal: %v", err)
	}

	used := usedByWeek(t, h, listingID)
	if used[weekKey(0)] != 0 {
		t.Fatalf("minggu ditandai penuh used = %d, mau 0 (dilewati)", used[weekKey(0)])
	}
	if used[weekKey(1)] != 500 {
		t.Fatalf("minggu berikutnya used = %d, mau 500", used[weekKey(1)])
	}
}

// TestReverseAllocation_RestoresAllPeriods_FR020 forms an order spanning three
// weeks, then reverses it: every period's used_capacity returns to zero and each
// allocation row is marked reversed_at rather than deleted (FR-020).
func TestReverseAllocation_RestoresAllPeriods_FR020(t *testing.T) {
	h := newAcceptHarness(t, "alloc_reverse")
	base := platform.WeekStart(acceptBaseTime)
	deadline := base.AddDate(0, 0, 7*4)

	listingID, subAcc := seedListing(t, h, "alfa", 500, deadline, deadline)
	subProf := subProfileID(t, h, subAcc)

	buyer := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyer@contoh.test", false), "Pembeli")
	req := seedRequest(t, h, buyer, 1200, deadline)
	offer := seedOfferedCandidate(t, h, req, listingID, subProf, 1_000_000)
	acc := buyerAccountOf(t, h, buyer)

	view, err := h.svc.accept(context.Background(), acc, offer)
	if err != nil {
		t.Fatalf("accept gagal: %v", err)
	}
	if got := usedCapacity(t, h, listingID); got != 1200 {
		t.Fatalf("used_capacity sebelum pembalikan = %d, mau 1200", got)
	}

	woID, _ := parseUUID(view.WorkOrderID)
	if err := h.svc.ReverseAllocation(context.Background(), woID); err != nil {
		t.Fatalf("ReverseAllocation gagal: %v", err)
	}

	if got := usedCapacity(t, h, listingID); got != 0 {
		t.Fatalf("used_capacity setelah pembalikan = %d, mau 0", got)
	}

	// Rows survive, all marked reversed, none deleted.
	var total, reversed int
	if err := h.pool.QueryRow(context.Background(),
		`SELECT count(*), count(reversed_at) FROM capacity_allocation WHERE work_order_id = $1`,
		woID).Scan(&total, &reversed); err != nil {
		t.Fatalf("hitung alokasi: %v", err)
	}
	if total != 3 {
		t.Fatalf("baris alokasi = %d, mau 3 (tidak dihapus)", total)
	}
	if reversed != total {
		t.Fatalf("baris reversed_at = %d, mau %d (semua ditandai)", reversed, total)
	}
}

// TestPropagateCapacity_LeavesAllocatedPeriodUntouched_FR089 accepts an order
// that allocates the first week, then propagates a new weekly capacity. The
// allocated week keeps its old total_capacity; the unallocated future weeks take
// the new one. Propagation must never disturb a week backing a running order
// (FR-089).
func TestPropagateCapacity_LeavesAllocatedPeriodUntouched_FR089(t *testing.T) {
	h := newAcceptHarness(t, "alloc_propagate")
	base := platform.WeekStart(acceptBaseTime)
	deadline := base.AddDate(0, 0, 7*3)

	listingID, subAcc := seedListing(t, h, "alfa", 500, deadline, deadline)
	subProf := subProfileID(t, h, subAcc)

	buyer := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyer@contoh.test", false), "Pembeli")
	req := seedRequest(t, h, buyer, 300, deadline)
	offer := seedOfferedCandidate(t, h, req, listingID, subProf, 1_000_000)
	acc := buyerAccountOf(t, h, buyer)

	if _, err := h.svc.accept(context.Background(), acc, offer); err != nil {
		t.Fatalf("accept gagal: %v", err)
	}

	now := h.clock.Now()
	q := sqlcgen.New(h.pool)
	if err := q.PropagateCapacityToFuturePeriods(context.Background(), sqlcgen.PropagateCapacityToFuturePeriodsParams{
		ListingID:     listingID,
		TotalCapacity: 800,
		WeekStart:     pgdate(base),
		UpdatedAt:     tstz(now),
	}); err != nil {
		t.Fatalf("propagate: %v", err)
	}

	cap := capByWeek(t, h, listingID)
	if cap[weekKey(0)] != 500 {
		t.Fatalf("minggu teralokasi total_capacity = %d, mau tetap 500 (FR-089)", cap[weekKey(0)])
	}
	if cap[weekKey(1)] != 800 || cap[weekKey(2)] != 800 {
		t.Fatalf("minggu belum teralokasi total_capacity = %d/%d, mau 800/800", cap[weekKey(1)], cap[weekKey(2)])
	}
}

// TestAllocation_TriggerRejectsBeforeReadiness_FR087 inserts an allocation row on
// a week earlier than the work order's readiness_week_start directly, so the
// database trigger (not the fill logic) is exercised: it must reject the write
// (FR-087).
func TestAllocation_TriggerRejectsBeforeReadiness_FR087(t *testing.T) {
	h := newAcceptHarness(t, "alloc_trigger_readiness")
	base := platform.WeekStart(acceptBaseTime)
	readiness := base.AddDate(0, 0, 7)

	listingID, subAcc := seedListing(t, h, "alfa", 500, readiness, readiness)
	subProf := subProfileID(t, h, subAcc)

	buyer := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyer@contoh.test", false), "Pembeli")
	buyerAcc := buyerAccountOf(t, h, buyer)
	_ = buyerAcc
	req := seedRequest(t, h, buyer, 100, readiness)
	offer := seedOfferedCandidate(t, h, req, listingID, subProf, 1_000_000)

	ctx := context.Background()
	// Load the offer to build a work order whose readiness week is week 1.
	orow, err := h.svc.queries().GetOfferForAccept(ctx, offer)
	if err != nil {
		t.Fatalf("baca offer: %v", err)
	}
	q := sqlcgen.New(h.pool)
	wo, err := q.InsertWorkOrder(ctx, sqlcgen.InsertWorkOrderParams{
		CandidateID:        orow.CandidateID,
		OfferID:            orow.OfferID,
		BuyerID:            orow.BuyerID,
		SubcontractorID:    orow.SubcontractorID,
		Quantity:           orow.Quantity,
		TotalPrice:         orow.TotalPrice,
		Deadline:           orow.Deadline,
		ReadinessWeekStart: pgdate(readiness),
		CreatedAt:          tstz(h.clock.Now()),
	})
	if err != nil {
		t.Fatalf("insert work order: %v", err)
	}

	// The base-week period precedes the readiness week; allocating on it must fail.
	var periodID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`SELECT id FROM availability_period WHERE listing_id = $1 AND week_start = $2`,
		listingID, base).Scan(&periodID); err != nil {
		t.Fatalf("baca period: %v", err)
	}

	_, err = q.InsertAllocation(ctx, sqlcgen.InsertAllocationParams{
		WorkOrderID: wo.ID,
		PeriodID:    periodID,
		Quantity:    50,
		CreatedAt:   tstz(h.clock.Now()),
	})
	if err == nil {
		t.Fatal("alokasi sebelum minggu kesiapan berhasil, mau ditolak trigger (FR-087)")
	}
}

// capByWeek returns a listing's total_capacity keyed by week_start date text.
func capByWeek(t *testing.T, h *acceptHarness, listingID pgtype.UUID) map[string]int32 {
	t.Helper()
	rows, err := h.pool.Query(context.Background(),
		`SELECT week_start, total_capacity FROM availability_period WHERE listing_id = $1`, listingID)
	if err != nil {
		t.Fatalf("baca periode: %v", err)
	}
	defer rows.Close()
	out := map[string]int32{}
	for rows.Next() {
		var wk pgtype.Date
		var cap int32
		if err := rows.Scan(&wk, &cap); err != nil {
			t.Fatalf("scan periode: %v", err)
		}
		out[wk.Time.Format("2006-01-02")] = cap
	}
	return out
}

var _ = time.Second
