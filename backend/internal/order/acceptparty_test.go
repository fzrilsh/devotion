package order

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// appendCounterOffer chains a further offer onto the candidate that owns
// prevOffer, attributed to the given party. It mirrors what the quota counter
// endpoint writes (a new row, never an update) so the accept guard can be
// exercised against a chain whose standing round belongs to either side.
func appendCounterOffer(t *testing.T, h *acceptHarness, prevOffer pgtype.UUID, by sqlcgen.OfferParty, price int64) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	var candidateID pgtype.UUID
	var seq int32
	if err := h.pool.QueryRow(ctx,
		`SELECT candidate_id, sequence FROM offer WHERE id = $1`, prevOffer).Scan(&candidateID, &seq); err != nil {
		t.Fatalf("baca penawaran sebelumnya: %v", err)
	}
	var next pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO offer (candidate_id, sequence, proposed_by, total_price, readiness_lead_days, created_at)
		 VALUES ($1, $2, $3, $4, 0, $5) RETURNING id`,
		candidateID, seq+1, by, price, acceptBaseTime).Scan(&next); err != nil {
		t.Fatalf("seed penawaran balik: %v", err)
	}
	return next
}

// TestAccept_ProposerCannotAcceptOwnOffer_FR033 proves closing a negotiation is
// the counterpart's approval, not the proposer's. Two chains are built, one whose
// standing round is the buyer's counter and one whose standing round is the
// subcontractor's offer, and in each the party who made that round is refused
// with FORBIDDEN. Without this guard a buyer could accept their own counter and
// bind the subcontractor to a price never agreed to, which is the invariant
// FR-033 rests on: rounds alternate until one side approves the other's.
func TestAccept_ProposerCannotAcceptOwnOffer_FR033(t *testing.T) {
	h := newAcceptHarness(t, "accept_own_offer")
	week := platform.WeekStart(acceptBaseTime)

	listingID, subAcc := seedListing(t, h, "alfa", 1000, week, week)
	subProf := subProfileID(t, h, subAcc)
	buyerProf := seedAcceptProfile(t, h.pool,
		seedAcceptAccount(t, h.pool, "buyer-own@contoh.test", false), "Pembeli")
	buyerAcc := buyerAccountOf(t, h, buyerProf)

	// Chain 1: subcontractor offers, buyer counters. The buyer owns the standing
	// round, so the buyer may not accept it.
	req1 := seedRequest(t, h, buyerProf, 50, week)
	seq1 := seedOfferedCandidate(t, h, req1, listingID, subProf, 150_000_000)
	buyerCounter := appendCounterOffer(t, h, seq1, sqlcgen.OfferPartyBuyer, 1_000_000)

	// Chain 2: subcontractor offers only. The subcontractor owns the standing
	// round, so the subcontractor may not accept it.
	req2 := seedRequest(t, h, buyerProf, 50, week)
	subOffer := seedOfferedCandidate(t, h, req2, listingID, subProf, 150_000_000)

	cases := []struct {
		name    string
		caller  pgtype.UUID
		offerID pgtype.UUID
	}{
		{"buyer menyetujui counter-nya sendiri", buyerAcc, buyerCounter},
		{"subkontraktor menyetujui penawarannya sendiri", subAcc, subOffer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := h.svc.accept(context.Background(), tc.caller, tc.offerID)
			if err == nil {
				t.Fatal("accept berhasil, mau ditolak: hanya pihak lawan yang boleh menyetujui (FR-033)")
			}
			if got := errCode(err); got != httpx.CodeForbidden {
				t.Fatalf("kode = %q, mau %q; galat = %v", got, httpx.CodeForbidden, err)
			}
		})
	}

	// Nothing was allocated and no order formed by either refused call.
	if got := usedCapacity(t, h, listingID); got != 0 {
		t.Fatalf("used_capacity = %d, mau 0: penolakan tidak boleh mengalokasikan apa pun", got)
	}
}

// TestAccept_SubcontractorAcceptsBuyerCounter_FR033 proves the capability spec
// skenario 4 describes exists: when the standing round is the buyer's counter,
// the subcontractor approves it and the agreement forms at that price. Before the
// fix the route admitted buyers only, so this chain could never be closed, the
// subcontractor could only counter again forever.
func TestAccept_SubcontractorAcceptsBuyerCounter_FR033(t *testing.T) {
	h := newAcceptHarness(t, "accept_sub_closes")
	week := platform.WeekStart(acceptBaseTime)

	listingID, subAcc := seedListing(t, h, "alfa", 100, week, week)
	subProf := subProfileID(t, h, subAcc)
	buyerProf := seedAcceptProfile(t, h.pool,
		seedAcceptAccount(t, h.pool, "buyer-sub-close@contoh.test", false), "Pembeli")

	req := seedRequest(t, h, buyerProf, 50, week)
	seq1 := seedOfferedCandidate(t, h, req, listingID, subProf, 150_000_000)
	const counterPrice = 100_000_000
	buyerCounter := appendCounterOffer(t, h, seq1, sqlcgen.OfferPartyBuyer, counterPrice)

	view, err := h.svc.accept(context.Background(), subAcc, buyerCounter)
	if err != nil {
		t.Fatalf("subkontraktor accept counter buyer gagal, mau berhasil (FR-033): %v", err)
	}
	if view.TotalPrice != counterPrice {
		t.Fatalf("total_price = %d, mau %d (harga ronde terakhir)", view.TotalPrice, counterPrice)
	}
	if view.Status != string(sqlcgen.WorkOrderStatusAccepted) {
		t.Fatalf("status = %q, mau %q", view.Status, sqlcgen.WorkOrderStatusAccepted)
	}
	if got := usedCapacity(t, h, listingID); got != 50 {
		t.Fatalf("used_capacity = %d, mau 50", got)
	}

	// The status history actor is the subcontractor, the party that accepted.
	var changedBy pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`SELECT h.changed_by FROM work_order_status_history h
		 JOIN work_order w ON w.id = h.work_order_id
		 WHERE w.offer_id = $1`, buyerCounter).Scan(&changedBy); err != nil {
		t.Fatalf("baca riwayat status: %v", err)
	}
	if changedBy != subAcc {
		t.Fatalf("changed_by = %v, mau akun subkontraktor %v", changedBy, subAcc)
	}
}

// TestAccept_RejectsNonParty_FR033 proves an account that is neither the buyer
// nor the listing owner gets a 404 rather than a hint that the offer exists.
// Opening the route to both business roles widens who clears the gate, so the
// party guard inside the service is what keeps a third party out.
func TestAccept_RejectsNonParty_FR033(t *testing.T) {
	h := newAcceptHarness(t, "accept_non_party")
	week := platform.WeekStart(acceptBaseTime)

	listingID, subAcc := seedListing(t, h, "alfa", 1000, week, week)
	buyerProf := seedAcceptProfile(t, h.pool,
		seedAcceptAccount(t, h.pool, "buyer-party@contoh.test", false), "Pembeli")
	req := seedRequest(t, h, buyerProf, 50, week)
	offer := seedOfferedCandidate(t, h, req, listingID, subProfileID(t, h, subAcc), 1_000_000)

	// A different subcontractor, holding the role but no stake in this chain.
	_, strangerAcc := seedListing(t, h, "beta", 1000, week, week)

	_, err := h.svc.accept(context.Background(), strangerAcc, offer)
	if err == nil {
		t.Fatal("accept oleh bukan pihak berhasil, mau ditolak")
	}
	if got := errCode(err); got != httpx.CodeNotFound {
		t.Fatalf("kode = %q, mau %q; galat = %v", got, httpx.CodeNotFound, err)
	}
}
