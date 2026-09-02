package order

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// adminRouter wires the order routes with an admin principal, so a test drives the
// three mediation routes over HTTP through their real role gate. The admin routes
// carry no party guard, so any admin account exercises them.
func adminRouter(t *testing.T, h *woHarness, email string) http.Handler {
	t.Helper()
	admin := seedAcceptAccount(t, h.pool, email, true)
	return woRouter(h, httpx.RoleAdmin, admin)
}

// listDisputesReq drives GET /api/admin/disputes with an optional raw query string
// (including the leading "?", or empty for none).
func listDisputesReq(handler http.Handler, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/disputes"+query, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// mediateReq drives POST /api/admin/disputes/{id}/mediate.
func mediateReq(handler http.Handler, disputeID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost,
		"/api/admin/disputes/"+disputeID+"/mediate", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// resolveReq drives POST /api/admin/disputes/{id}/resolve with the given body.
func resolveReq(handler http.Handler, disputeID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost,
		"/api/admin/disputes/"+disputeID+"/resolve", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// seedOpenDispute reports a dispute on the harness order through the buyer route,
// then reads the created dispute's id from the pool. The report leaves the order on
// its current status; only admin mediation moves it.
func seedOpenDispute(t *testing.T, h *woHarness) string {
	t.Helper()
	buyer := woRouter(h, httpx.RoleBuyer, h.buyerAcc)
	rec := disputeReq(buyer, uuidString(h.workOrderID),
		`{"report_body":"Barang tidak sesuai contoh yang disepakati."}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("buka sengketa: status %d, body %s", rec.Code, rec.Body.String())
	}
	var id pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`SELECT id FROM dispute WHERE work_order_id = $1`, h.workOrderID).Scan(&id); err != nil {
		t.Fatalf("baca dispute id: %v", err)
	}
	return uuidString(id)
}

// decodeDisputeView parses a Dispute response body.
func decodeDisputeView(t *testing.T, rec *httptest.ResponseRecorder) disputeView {
	t.Helper()
	var v disputeView
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode Dispute %q: %v", rec.Body.String(), err)
	}
	return v
}

// FR-046: "Salah satu pihak MUST dapat melaporkan sengketa atas sebuah pesanan,
// dan admin MUST dapat menandai pesanan tersebut \"Dalam Mediasi\" serta melihat
// seluruh riwayat request, penawaran, status, alokasi kapasitas, catatan
// pembayaran, alasan pembatalan, dan lampirannya."

// TestMediation_AdminListsDisputeQueue_FR046 proves an admin reads the dispute
// queue as a bare Dispute array and sees the reported dispute in it.
func TestMediation_AdminListsDisputeQueue_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_list")
	disputeID := seedOpenDispute(t, h)
	admin := adminRouter(t, h, "admin_med_list@contoh.test")

	rec := listDisputesReq(admin, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var items []disputeView
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("queue bukan array Dispute telanjang %q: %v", rec.Body.String(), err)
	}
	if len(items) != 1 || items[0].DisputeID != disputeID {
		t.Fatalf("queue = %+v, mau satu sengketa %q", items, disputeID)
	}
	if items[0].Status != string(sqlcgen.DisputeStatusReported) {
		t.Fatalf("status sengketa %q, mau %q", items[0].Status, sqlcgen.DisputeStatusReported)
	}
}

// TestMediation_ListRejectsNonAdmin_FR046 proves the queue's role gate turns a
// non-admin (a buyer) away with 403 before the handler runs.
func TestMediation_ListRejectsNonAdmin_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_list_role")
	seedOpenDispute(t, h)
	buyer := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	rec := listDisputesReq(buyer, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403; body %s", rec.Code, rec.Body.String())
	}
}

// TestMediation_ListRejectsBadStatusFilter_FR046 proves an unknown status filter
// value is invalid input (422 VALIDATION_FAILED), not silently ignored, so a typo
// does not quietly widen the queue.
func TestMediation_ListRejectsBadStatusFilter_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_list_badfilter")
	seedOpenDispute(t, h)
	admin := adminRouter(t, h, "admin_med_badfilter@contoh.test")

	rec := listDisputesReq(admin, "?status=bogus")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeValidationFailed)
	}
}

// TestMediation_AdminMovesToMediation_FR046 proves an admin marks the reported
// dispute's order "Dalam Mediasi": the dispute comes back in_mediation and the work
// order status is in_mediation, moved by a system history row.
func TestMediation_AdminMovesToMediation_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_mediate")
	shipWorkOrder(t, h)
	disputeID := seedOpenDispute(t, h)
	admin := adminRouter(t, h, "admin_med_mediate@contoh.test")

	rec := mediateReq(admin, disputeID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	if got := decodeDisputeView(t, rec).Status; got != string(sqlcgen.DisputeStatusInMediation) {
		t.Fatalf("status sengketa %q, mau %q (FR-046)", got, sqlcgen.DisputeStatusInMediation)
	}
	wo := loadWorkOrder(t, h)
	if wo.Status != sqlcgen.WorkOrderStatusInMediation {
		t.Fatalf("status pesanan %q, mau %q (FR-046)", wo.Status, sqlcgen.WorkOrderStatusInMediation)
	}
}

// TestMediation_MediateRejectsNonAdmin_FR046 proves the mediate action's role gate
// turns a non-admin (the subcontractor) away with 403 before the handler runs.
func TestMediation_MediateRejectsNonAdmin_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_mediate_role")
	disputeID := seedOpenDispute(t, h)
	sub := woRouter(h, httpx.RoleSubcontractor, h.subAcc)

	rec := mediateReq(sub, disputeID)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403; body %s", rec.Code, rec.Body.String())
	}
}

// FR-067: "Admin MUST dapat menutup pesanan yang berada dalam mediasi sebagai
// dibatalkan dengan catatan, dan pada saat itu MUST menentukan secara eksplisit
// apakah seluruh alokasi kapasitasnya dibalik dan pihak mana yang menanggung
// pembatalan tersebut dalam perhitungan tingkat penyelesaian."

// FR-072: "Sebuah pesanan yang dibatalkan MUST masuk pembagi tingkat penyelesaian
// hanya bagi pihak yang membatalkan, dan MUST TIDAK mempengaruhi tingkat
// penyelesaian pihak lain sama sekali."

// mediateThenResolve mediates the open dispute then resolves it with the given
// body, returning the resolve recorder. It fails the test if mediation does not
// take.
func mediateThenResolve(t *testing.T, admin http.Handler, disputeID, body string) *httptest.ResponseRecorder {
	t.Helper()
	if rec := mediateReq(admin, disputeID); rec.Code != http.StatusOK {
		t.Fatalf("mediasi: status %d, body %s", rec.Code, rec.Body.String())
	}
	return resolveReq(admin, disputeID, body)
}

// TestMediation_ResolveCancelledChargesLiableParty_FR067_FR072 proves the admin
// closes a mediated order as cancelled with an explicit liable party and allocation
// choice: the order is cancelled, cancelled_by_id is the liable profile (so only
// that party carries the cancellation in the completion rate, FR-072), and the
// dispute stores the admin's allocation_reversed and liable_profile_id (FR-067).
func TestMediation_ResolveCancelledChargesLiableParty_FR067_FR072(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_cancel")
	disputeID := seedOpenDispute(t, h)
	admin := adminRouter(t, h, "admin_med_cancel@contoh.test")

	// The subcontractor's business profile bears the cancellation.
	liable := subProfileID(t, h.acceptHarness, h.subAcc)
	body := `{"result":"cancelled","allocation_reversed":true,` +
		`"liable_profile_id":"` + uuidString(liable) + `",` +
		`"note":"Barang tidak dikirim setelah tenggat, subkontraktor menanggung."}`

	rec := mediateThenResolve(t, admin, disputeID, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	view := decodeDisputeView(t, rec)
	if view.Status != string(sqlcgen.DisputeStatusResolved) {
		t.Fatalf("status sengketa %q, mau %q (FR-067)", view.Status, sqlcgen.DisputeStatusResolved)
	}
	if view.Result == nil || *view.Result != string(sqlcgen.DisputeResultCancelled) {
		t.Fatalf("result = %v, mau %q (FR-067)", view.Result, sqlcgen.DisputeResultCancelled)
	}
	if view.AllocationReversed == nil || !*view.AllocationReversed {
		t.Fatalf("allocation_reversed = %v, mau true (FR-067)", view.AllocationReversed)
	}
	if view.LiableProfileID == nil || *view.LiableProfileID != uuidString(liable) {
		t.Fatalf("liable_profile_id = %v, mau %q (FR-067)", view.LiableProfileID, uuidString(liable))
	}

	// The order is cancelled and cancelled_by_id is the liable profile, so the
	// completion-rate divisor charges only that party (FR-072).
	wo := loadWorkOrder(t, h)
	if wo.Status != sqlcgen.WorkOrderStatusCancelled {
		t.Fatalf("status pesanan %q, mau %q (FR-072)", wo.Status, sqlcgen.WorkOrderStatusCancelled)
	}
	if wo.CancelledByID != liable {
		t.Fatalf("cancelled_by_id bukan pihak yang menanggung; tingkat penyelesaian akan membebani pihak yang salah (FR-072)")
	}
}

// TestMediation_ResolveCancelledRequiresLiableParty_FR067 proves the cancelled
// branch rejects a body with no liable_profile_id as invalid input (422): the admin
// MUST name which party bears the cancellation, it is not derivable.
func TestMediation_ResolveCancelledRequiresLiableParty_FR067(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_cancel_noliable")
	disputeID := seedOpenDispute(t, h)
	admin := adminRouter(t, h, "admin_med_noliable@contoh.test")

	body := `{"result":"cancelled","allocation_reversed":true,"note":"tidak lengkap"}`
	rec := mediateThenResolve(t, admin, disputeID, body)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeValidationFailed)
	}
}

// TestMediation_ResolveCancelledRequiresNote_FR067 proves the cancelled branch
// rejects a body with an empty note as invalid input (422): a cancellation must
// carry its reason, so the admin cannot close a case without one even when the
// liable party is named.
func TestMediation_ResolveCancelledRequiresNote_FR067(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_cancel_nonote")
	disputeID := seedOpenDispute(t, h)
	admin := adminRouter(t, h, "admin_med_nonote@contoh.test")

	liable := subProfileID(t, h.acceptHarness, h.subAcc)
	body := `{"result":"cancelled","allocation_reversed":true,` +
		`"liable_profile_id":"` + uuidString(liable) + `","note":""}`
	rec := mediateThenResolve(t, admin, disputeID, body)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeValidationFailed)
	}
}

// TestMediation_ResolveRejectsBadResult_FR067 proves an unknown result enum is
// invalid input (422 VALIDATION_FAILED).
func TestMediation_ResolveRejectsBadResult_FR067(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_badresult")
	disputeID := seedOpenDispute(t, h)
	admin := adminRouter(t, h, "admin_med_badresult@contoh.test")

	rec := mediateThenResolve(t, admin, disputeID, `{"result":"bogus"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeValidationFailed)
	}
}

// TestMediation_ResolveRejectsOverlongNote_FR067 proves a note over 2000 characters
// is invalid input (422 VALIDATION_FAILED).
func TestMediation_ResolveRejectsOverlongNote_FR067(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_longnote")
	disputeID := seedOpenDispute(t, h)
	admin := adminRouter(t, h, "admin_med_longnote@contoh.test")

	body := `{"result":"confirmed","note":"` + strings.Repeat("a", 2001) + `"}`
	rec := mediateThenResolve(t, admin, disputeID, body)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeValidationFailed)
	}
}

// TestMediation_ResolveRejectsNonAdmin_FR067 proves the resolve action's role gate
// turns a non-admin (the buyer) away with 403 before the handler runs.
func TestMediation_ResolveRejectsNonAdmin_FR067(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_resolve_role")
	disputeID := seedOpenDispute(t, h)
	buyer := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	rec := resolveReq(buyer, disputeID, `{"result":"confirmed","note":"x"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403; body %s", rec.Code, rec.Body.String())
	}
}

// FR-068: "Sistem MUST menandai pesanan berstatus \"Dikirim\" sebagai dikonfirmasi
// diterima secara otomatis setelah 7 hari sejak status tersebut ditetapkan, dan
// MUST memberi tahu kedua pihak bahwa penutupan terjadi secara otomatis."

// TestMediation_ContinuedRestartsAutoConfirmClock_FR068 is the test that proves the
// core mediation decision works: a shipped order (day 0) whose dispute is reported
// (day 3) and mediated then resolved 'continued' (day 20) must NOT auto-confirm on
// the next tick, because the 7-day auto-confirm clock restarts from the mediation
// close (day 20), not the original shipment. The Clock is advanced, never waited on.
// Without the restart the order would be 20 days past its original shipment and the
// ticker would close it immediately; with the restart the new deadline is day 27,
// so a tick at day 21 leaves it shipped and a tick past day 27 finally closes it.
func TestMediation_ContinuedRestartsAutoConfirmClock_FR068(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_med_continued")
	shipWorkOrder(t, h) // day 0, clock at acceptBaseTime

	// Day 3: a party reports a dispute, halting the original auto-confirm count.
	h.clock.Advance(3 * 24 * time.Hour)
	disputeID := seedOpenDispute(t, h)

	// Day 20: admin mediates then resolves 'continued'. The order returns to shipped
	// and the auto-confirm clock restarts from now (day 20).
	h.clock.Advance(17 * 24 * time.Hour)
	admin := adminRouter(t, h, "admin_med_continued@contoh.test")
	resolveClose := h.clock.Now()
	rec := mediateThenResolve(t, admin, disputeID,
		`{"result":"continued","note":"Sengketa selesai, pesanan dilanjutkan."}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("resolve continued: status %d, body %s", rec.Code, rec.Body.String())
	}
	if got := decodeDisputeView(t, rec).Result; got == nil || *got != string(sqlcgen.DisputeResultContinued) {
		t.Fatalf("result = %v, mau %q", got, sqlcgen.DisputeResultContinued)
	}

	// The order is back to shipped, not confirmed.
	wo := loadWorkOrder(t, h)
	if wo.Status != sqlcgen.WorkOrderStatusShipped {
		t.Fatalf("status pesanan %q, mau %q; mediasi 'continued' mengembalikan ke dikirim", wo.Status, sqlcgen.WorkOrderStatusShipped)
	}

	// Day 21: one day after the mediation close, well past the ORIGINAL 7-day
	// window (day 7) but inside the RESTARTED window (day 27). The ticker must NOT
	// close it: this is what proves the clock restarted from day 20, not day 0.
	h.clock.Set(resolveClose.Add(24 * time.Hour))
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm sebelum tenggat baru: %v", err)
	}
	if loadWorkOrder(t, h).AutoConfirmed {
		t.Fatal("pesanan terkonfirmasi otomatis di hari ke-21; jam konfirmasi harus mulai ulang dari penutupan mediasi hari ke-20, bukan pengiriman asli (FR-068)")
	}

	// Just past the restarted 7-day window (day 20 + 7 + a bit): now it closes,
	// confirming the new deadline runs from day 20.
	h.clock.Set(resolveClose.Add(AutoConfirmWindow + time.Hour))
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm setelah tenggat baru: %v", err)
	}
	closed := loadWorkOrder(t, h)
	if closed.Status != sqlcgen.WorkOrderStatusConfirmed || !closed.AutoConfirmed {
		t.Fatalf("pesanan tidak tertutup setelah tenggat baru (hari ke-27): status %q auto_confirmed %v; tenggat baru harus 7 hari dari penutupan mediasi (FR-068)",
			closed.Status, closed.AutoConfirmed)
	}
}
