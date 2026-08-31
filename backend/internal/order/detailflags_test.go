package order

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
)

// viewFor reads the fresh detail row and builds the view the given caller would
// see, so a flag test asserts exactly what the frontend receives.
func viewFor(t *testing.T, h *woHarness, caller pgtype.UUID) workOrderView {
	t.Helper()
	row, err := h.svc.queries().GetWorkOrderForView(context.Background(), h.workOrderID)
	if err != nil {
		t.Fatalf("GetWorkOrderForView: %v", err)
	}
	view, err := h.svc.buildDetailView(context.Background(), row, caller)
	if err != nil {
		t.Fatalf("buildDetailView: %v", err)
	}
	return view
}

// TestDetailFlags_AcceptedPartyCanPayNotReview_FR041_FR047 proves a fresh accepted
// order offers the payment button to a party (can_record_payment) but not the
// review button (can_review), because a review needs a confirmed-received order,
// and shows no payment mismatch when neither side has stated.
func TestDetailFlags_AcceptedPartyCanPayNotReview_FR041_FR047(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_flags_accepted")

	view := viewFor(t, h, h.buyerAcc)
	if !view.CanRecordPayment {
		t.Fatal("can_record_payment false pada pesanan accepted; pihak boleh mencatat pembayaran (FR-041)")
	}
	if view.CanReview {
		t.Fatal("can_review true sebelum pesanan dikonfirmasi; ulasan hanya untuk pesanan selesai (FR-047)")
	}
	if view.PaymentMismatch != nil {
		t.Fatalf("payment_mismatch %+v, mau nil ketika belum ada pernyataan (FR-043)", view.PaymentMismatch)
	}
}

// TestDetailFlags_AdminGetsNeitherButton_FR041_FR047 proves an admin reading an
// order it is on neither side of gets can_record_payment and can_review both
// false: a payment statement and a review are actions of the two parties, so the
// client offers the admin neither button even though the admin can read the order.
func TestDetailFlags_AdminGetsNeitherButton_FR041_FR047(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_flags_admin")
	admin := seedAcceptAccount(t, h.pool, "admin_flags@contoh.test", true)

	view := viewFor(t, h, admin)
	if view.CanRecordPayment {
		t.Fatal("can_record_payment true untuk admin bukan pihak; hanya pihak yang mencatat pembayaran (FR-041)")
	}
	if view.CanReview {
		t.Fatal("can_review true untuk admin bukan pihak; hanya pihak yang memberi ulasan (FR-047)")
	}
}

// TestDetailFlags_CancelledOrderNotPayable_FR041 proves a cancelled order turns
// off can_record_payment for a party: a cancelled order settles no money through
// the platform, so the payment button is withdrawn.
func TestDetailFlags_CancelledOrderNotPayable_FR041(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_flags_cancelled")

	if _, err := h.svc.cancel(context.Background(), h.buyerAcc, h.workOrderID,
		"Pembeli membatalkan sebelum produksi dimulai."); err != nil {
		t.Fatalf("batalkan pesanan: %v", err)
	}

	view := viewFor(t, h, h.buyerAcc)
	if view.Status != string(sqlcgen.WorkOrderStatusCancelled) {
		t.Fatalf("status %q, mau %q", view.Status, sqlcgen.WorkOrderStatusCancelled)
	}
	if view.CanRecordPayment {
		t.Fatal("can_record_payment true pada pesanan cancelled; pembayaran tidak dicatat di pesanan batal (FR-041)")
	}
}

// TestDetailFlags_ConfirmedPartyCanReviewOnce_FR047 proves a confirmed order offers
// the review button to a party that has not yet reviewed, and withdraws it once
// that party has reviewed, matching the one_review_per_order_per_reviewer leg the
// review endpoint enforces.
func TestDetailFlags_ConfirmedPartyCanReviewOnce_FR047(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_flags_review")
	shipWorkOrder(t, h)
	if _, err := h.svc.confirm(context.Background(), h.buyerAcc, h.workOrderID); err != nil {
		t.Fatalf("konfirmasi pesanan: %v", err)
	}

	// Before reviewing, the buyer sees the review button.
	before := viewFor(t, h, h.buyerAcc)
	if before.Status != string(sqlcgen.WorkOrderStatusConfirmed) {
		t.Fatalf("status %q, mau %q", before.Status, sqlcgen.WorkOrderStatusConfirmed)
	}
	if !before.CanReview {
		t.Fatal("can_review false pada pesanan confirmed yang belum diulas pemanggil (FR-047)")
	}

	// The buyer submits a review, so the button must withdraw on the next read.
	buyerProfile, err := h.svc.queries().GetProfileIDByAccount(context.Background(), h.buyerAcc)
	if err != nil {
		t.Fatalf("resolusi profil buyer: %v", err)
	}
	subProfile, err := h.svc.queries().GetProfileIDByAccount(context.Background(), h.subAcc)
	if err != nil {
		t.Fatalf("resolusi profil subkontraktor: %v", err)
	}
	if _, err := h.pool.Exec(context.Background(),
		`INSERT INTO review (work_order_id, reviewer_id, reviewee_id, rating, created_at)
		 VALUES ($1, $2, $3, 5, $4)`,
		h.workOrderID, buyerProfile, subProfile, h.clock.Now()); err != nil {
		t.Fatalf("simpan ulasan: %v", err)
	}

	after := viewFor(t, h, h.buyerAcc)
	if after.CanReview {
		t.Fatal("can_review true setelah pemanggil mengulas; tiap pihak mengulas sekali per pesanan (FR-047)")
	}
	// The subcontractor has not reviewed yet, so their button is still offered.
	subView := viewFor(t, h, h.subAcc)
	if !subView.CanReview {
		t.Fatal("can_review false untuk subkontraktor yang belum mengulas (FR-047)")
	}
}

// TestDetailFlags_MissingCounterpart_FR043 proves that when one party has stated a
// payment and the other has not stated at all, payment_mismatch flags
// missing_counterpart for every reader, caller-independent.
func TestDetailFlags_MissingCounterpart_FR043(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_flags_missing")

	if _, err := h.svc.createPayment(context.Background(), h.buyerAcc, h.workOrderID,
		sqlcgen.PaymentDirectionSent, mustDate(t, "2026-08-20"), nil); err != nil {
		t.Fatalf("catat pernyataan buyer: %v", err)
	}

	// Same result for both parties and admin: the flag does not depend on the reader.
	admin := seedAcceptAccount(t, h.pool, "admin_mismatch@contoh.test", true)
	for _, caller := range []pgtype.UUID{h.buyerAcc, h.subAcc, admin} {
		view := viewFor(t, h, caller)
		if view.PaymentMismatch == nil {
			t.Fatal("payment_mismatch nil padahal satu pihak belum menyatakan (FR-043)")
		}
		if view.PaymentMismatch.Kind != "missing_counterpart" {
			t.Fatalf("kind %q, mau %q (FR-043)", view.PaymentMismatch.Kind, "missing_counterpart")
		}
	}
}

// TestDetailFlags_DateDiffersAndAgrees_FR043 proves payment_mismatch flags
// date_differs with the day gap when both parties stated on different dates, and
// clears to nil when both stated the same date.
func TestDetailFlags_DateDiffersAndAgrees_FR043(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_flags_datediff")

	if _, err := h.svc.createPayment(context.Background(), h.buyerAcc, h.workOrderID,
		sqlcgen.PaymentDirectionSent, mustDate(t, "2026-08-20"), nil); err != nil {
		t.Fatalf("catat pernyataan buyer: %v", err)
	}
	if _, err := h.svc.createPayment(context.Background(), h.subAcc, h.workOrderID,
		sqlcgen.PaymentDirectionReceived, mustDate(t, "2026-08-23"), nil); err != nil {
		t.Fatalf("catat pernyataan subkontraktor: %v", err)
	}

	view := viewFor(t, h, h.buyerAcc)
	if view.PaymentMismatch == nil {
		t.Fatal("payment_mismatch nil padahal kedua tanggal berbeda (FR-043)")
	}
	if view.PaymentMismatch.Kind != "date_differs" {
		t.Fatalf("kind %q, mau %q (FR-043)", view.PaymentMismatch.Kind, "date_differs")
	}
	if view.PaymentMismatch.DayDifference == nil || *view.PaymentMismatch.DayDifference != 3 {
		t.Fatalf("day_difference %v, mau 3 (FR-043)", view.PaymentMismatch.DayDifference)
	}
}

// TestDetailFlags_SameDateNoMismatch_FR043 proves two statements on the same date
// leave payment_mismatch nil: agreeing dates are no contradiction.
func TestDetailFlags_SameDateNoMismatch_FR043(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_flags_samedate")

	if _, err := h.svc.createPayment(context.Background(), h.buyerAcc, h.workOrderID,
		sqlcgen.PaymentDirectionSent, mustDate(t, "2026-08-20"), nil); err != nil {
		t.Fatalf("catat pernyataan buyer: %v", err)
	}
	if _, err := h.svc.createPayment(context.Background(), h.subAcc, h.workOrderID,
		sqlcgen.PaymentDirectionReceived, mustDate(t, "2026-08-20"), nil); err != nil {
		t.Fatalf("catat pernyataan subkontraktor: %v", err)
	}

	view := viewFor(t, h, h.buyerAcc)
	if view.PaymentMismatch != nil {
		t.Fatalf("payment_mismatch %+v, mau nil ketika tanggal sama (FR-043)", view.PaymentMismatch)
	}
}

// mustDate parses a YYYY-MM-DD date for a test, failing the test on a bad literal.
func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("tanggal uji tidak sah %q: %v", s, err)
	}
	return d
}
