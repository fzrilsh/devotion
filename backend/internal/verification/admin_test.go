package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// asAdmin flips the harness principal to a freshly seeded admin account and
// returns its id, so a test can drive the admin-gated queue and decision routes.
func asAdmin(t *testing.T, h *harness, email string) pgtype.UUID {
	t.Helper()
	adminAcc := seedAdmin(t, h.pool, email)
	h.asPrincipal(&httpx.Principal{
		Roles:   httpx.RoleAdmin,
		Account: sqlcgen.UserAccount{ID: adminAcc},
	})
	return adminAcc
}

// submitAsBuyer uploads the two files and submits a pending request as the
// harness's buyer principal, returning the created request id. It leaves the
// principal set to the buyer; a caller flips to admin afterward.
func submitAsBuyer(t *testing.T, h *harness, number string) string {
	t.Helper()
	idFile, locFile := uploadTwo(t, h)
	rec := h.doJSON("POST", "/api/verification", submitBody(number, idFile, locFile))
	mustStatus(t, rec, http.StatusCreated)
	var got verificationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode submit: %v", err)
	}
	return got.RequestID
}

// profileVerified reads the verified flag straight from business_profile so a
// test can prove an approval flipped the badge (which search.sql already selects,
// so the badge reaches search results too, FR-008).
func profileVerified(t *testing.T, h *harness, profile pgtype.UUID) bool {
	t.Helper()
	var v bool
	if err := h.pool.QueryRow(context.Background(),
		`SELECT verified FROM business_profile WHERE id = $1`, profile).Scan(&v); err != nil {
		t.Fatalf("read verified: %v", err)
	}
	return v
}

// TestQueue_ListsPending_FR007 proves the admin queue returns submitted requests
// with the business name and the pagination envelope, so an admin can work the
// backlog (FR-007).
func TestQueue_ListsPending_FR007(t *testing.T) {
	h := newHarness(t, "admin_queue_list")
	submitAsBuyer(t, h, "3273010101010001")

	asAdmin(t, h, "admin@contoh.test")
	rec := h.do("GET", "/api/admin/verification")
	mustStatus(t, rec, http.StatusOK)

	var got queueResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode queue: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("items %d, mau 1", len(got.Items))
	}
	if got.Items[0].BusinessName == "" {
		t.Fatal("business_name kosong di antrean")
	}
	if got.Items[0].Status != string(sqlcgen.VerificationStatusPending) {
		t.Fatalf("status %q, mau pending", got.Items[0].Status)
	}
	if got.Pagination.HasNext {
		t.Fatal("has_next true padahal hanya satu pengajuan")
	}
}

// TestQueue_StatusFilter_FR007 proves the status query param narrows the queue:
// a pending filter omits an already-decided request.
func TestQueue_StatusFilter_FR007(t *testing.T) {
	h := newHarness(t, "admin_queue_filter")
	reqID := submitAsBuyer(t, h, "3273010101010001")

	asAdmin(t, h, "admin@contoh.test")
	mustStatus(t, h.doJSON("POST", "/api/admin/verification/"+reqID+"/decision",
		`{"decision":"approved"}`), http.StatusOK)

	pending := h.do("GET", "/api/admin/verification?status=pending")
	mustStatus(t, pending, http.StatusOK)
	var got queueResponse
	if err := json.Unmarshal(pending.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Items) != 0 {
		t.Fatalf("items %d, mau 0 (yang disetujui tidak pending)", len(got.Items))
	}

	approved := h.do("GET", "/api/admin/verification?status=approved")
	mustStatus(t, approved, http.StatusOK)
	if err := json.Unmarshal(approved.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode approved: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("items %d, mau 1 pada filter approved", len(got.Items))
	}
}

// TestQueue_Keyset_FR007 proves the queue pages with a stable opaque cursor:
// requesting one item at a time walks every request without repeats or gaps.
func TestQueue_Keyset_FR007(t *testing.T) {
	h := newHarness(t, "admin_queue_keyset")
	// Three profiles, three pending requests.
	for i := 0; i < 3; i++ {
		acc, _ := seedProfile(t, h.pool, fmt.Sprintf("b%d@contoh.test", i), false)
		h.asPrincipal(&httpx.Principal{Roles: httpx.RoleBuyer, Account: sqlcgen.UserAccount{ID: acc}})
		submitAsBuyer(t, h, fmt.Sprintf("327301010101000%d", i))
	}
	// The default harness buyer also submits, four total.
	h.asPrincipal(&httpx.Principal{Roles: httpx.RoleBuyer, Account: sqlcgen.UserAccount{ID: h.acc}})
	submitAsBuyer(t, h, "3273010101019999")

	asAdmin(t, h, "admin@contoh.test")
	seen := map[string]bool{}
	cursor := ""
	for pages := 0; pages < 10; pages++ {
		path := "/api/admin/verification?status=pending"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		rec := h.do("GET", path)
		mustStatus(t, rec, http.StatusOK)
		var got queueResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode page: %v", err)
		}
		for _, it := range got.Items {
			if seen[it.RequestID] {
				t.Fatalf("request %s muncul dua kali antar halaman", it.RequestID)
			}
			seen[it.RequestID] = true
		}
		if !got.Pagination.HasNext {
			break
		}
		if got.Pagination.NextCursor == nil {
			t.Fatal("has_next true tapi next_cursor nil")
		}
		cursor = *got.Pagination.NextCursor
	}
	if len(seen) != 4 {
		t.Fatalf("terlihat %d pengajuan unik, mau 4", len(seen))
	}
}

// TestDecide_Approve_GrantsBadge_FR007_FR008 proves an approval records the
// decision (status, admin identity, time) and flips the verified badge on the
// profile, which search.sql already selects so the badge reaches search too.
func TestDecide_Approve_GrantsBadge_FR007_FR008(t *testing.T) {
	h := newHarness(t, "admin_decide_approve")
	reqID := submitAsBuyer(t, h, "3273010101010001")
	if profileVerified(t, h, h.profile) {
		t.Fatal("profil sudah terverifikasi sebelum keputusan")
	}

	adminAcc := asAdmin(t, h, "admin@contoh.test")
	rec := h.doJSON("POST", "/api/admin/verification/"+reqID+"/decision", `{"decision":"approved"}`)
	mustStatus(t, rec, http.StatusOK)

	var got verificationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != string(sqlcgen.VerificationStatusApproved) {
		t.Fatalf("status %q, mau approved", got.Status)
	}
	if got.DecidedAt == nil {
		t.Fatal("decided_at nil, keputusan harus tercatat waktunya")
	}
	if !profileVerified(t, h, h.profile) {
		t.Fatal("verified belum true setelah persetujuan")
	}

	// The decision recorded the deciding admin.
	var decidedBy pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`SELECT decided_by FROM verification_request WHERE id = $1`, reqID).Scan(&decidedBy); err != nil {
		t.Fatalf("read decided_by: %v", err)
	}
	if decidedBy != adminAcc {
		t.Fatalf("decided_by %v, mau admin %v", decidedBy, adminAcc)
	}

	// The applicant is told their request was decided (FR-051).
	if got := h.notif.countFor(h.acc, sqlcgen.EventTypeVerificationDecision); got != 1 {
		t.Fatalf("pemohon diberi tahu %d kali atas persetujuan, mau 1 (FR-051)", got)
	}
}

// TestDecide_RejectRecordsReason_FR007 proves a rejection records the reason the
// applicant reads and does NOT grant the badge, leaving the listing untouched
// (FR-010, FR-011).
func TestDecide_RejectRecordsReason_FR007(t *testing.T) {
	h := newHarness(t, "admin_decide_reject")
	reqID := submitAsBuyer(t, h, "3273010101010001")

	asAdmin(t, h, "admin@contoh.test")
	rec := h.doJSON("POST", "/api/admin/verification/"+reqID+"/decision",
		`{"decision":"rejected","reason":"Foto lokasi tidak jelas."}`)
	mustStatus(t, rec, http.StatusOK)

	var got verificationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != string(sqlcgen.VerificationStatusRejected) {
		t.Fatalf("status %q, mau rejected", got.Status)
	}
	if got.Reason == nil || *got.Reason != "Foto lokasi tidak jelas." {
		t.Fatalf("reason %v, mau alasan penolakan", got.Reason)
	}
	if profileVerified(t, h, h.profile) {
		t.Fatal("verified true padahal ditolak")
	}

	// The applicant is told the outcome even on a rejection (FR-051).
	if got := h.notif.countFor(h.acc, sqlcgen.EventTypeVerificationDecision); got != 1 {
		t.Fatalf("pemohon diberi tahu %d kali atas penolakan, mau 1 (FR-051)", got)
	}
}

// TestDecide_RejectWithoutReason_FR007 proves the handler refuses a reasonless
// rejection with a validation error before the rejection_needs_reason CHECK
// fires, so the applicant reads a field message rather than a 500.
func TestDecide_RejectWithoutReason_FR007(t *testing.T) {
	h := newHarness(t, "admin_decide_noreason")
	reqID := submitAsBuyer(t, h, "3273010101010001")

	asAdmin(t, h, "admin@contoh.test")
	rec := h.doJSON("POST", "/api/admin/verification/"+reqID+"/decision", `{"decision":"rejected"}`)
	mustStatus(t, rec, http.StatusUnprocessableEntity)
	p := decodeProblem(t, rec)
	if p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %s", p.Code, httpx.CodeValidationFailed)
	}
	if len(p.Errors) == 0 || p.Errors[0].Field != "reason" {
		t.Fatalf("errors %v, mau field reason", p.Errors)
	}
	// The request stays pending.
	pending := h.do("GET", "/api/admin/verification?status=pending")
	var got queueResponse
	_ = json.Unmarshal(pending.Body.Bytes(), &got)
	if len(got.Items) != 1 {
		t.Fatalf("items %d, mau 1 (penolakan gagal, tetap pending)", len(got.Items))
	}
}

// TestDecide_AlreadyDecided_FR007 proves a second decision on a decided request
// is refused rather than overwriting the first.
func TestDecide_AlreadyDecided_FR007(t *testing.T) {
	h := newHarness(t, "admin_decide_twice")
	reqID := submitAsBuyer(t, h, "3273010101010001")

	asAdmin(t, h, "admin@contoh.test")
	mustStatus(t, h.doJSON("POST", "/api/admin/verification/"+reqID+"/decision",
		`{"decision":"approved"}`), http.StatusOK)

	second := h.doJSON("POST", "/api/admin/verification/"+reqID+"/decision",
		`{"decision":"rejected","reason":"berubah pikiran"}`)
	mustStatus(t, second, http.StatusUnprocessableEntity)
}

// TestDecide_UnknownRequest_FR007 proves a decision on a nonexistent request id
// is a 404, not a 500.
func TestDecide_UnknownRequest_FR007(t *testing.T) {
	h := newHarness(t, "admin_decide_unknown")
	asAdmin(t, h, "admin@contoh.test")
	rec := h.doJSON("POST", "/api/admin/verification/00000000-0000-0000-0000-000000000000/decision",
		`{"decision":"approved"}`)
	mustStatus(t, rec, http.StatusNotFound)
}

// TestQueue_NonAdminRejected_FR007 proves a business caller is refused the admin
// queue with 403.
func TestQueue_NonAdminRejected_FR007(t *testing.T) {
	h := newHarness(t, "admin_queue_forbidden")
	rec := h.do("GET", "/api/admin/verification")
	mustStatus(t, rec, http.StatusForbidden)
}

// TestDecide_NonAdminRejected_FR007 proves a business caller is refused the
// decision route with 403 before any row is touched.
func TestDecide_NonAdminRejected_FR007(t *testing.T) {
	h := newHarness(t, "admin_decide_forbidden")
	reqID := submitAsBuyer(t, h, "3273010101010001")
	// Principal is still the buyer.
	rec := h.doJSON("POST", "/api/admin/verification/"+reqID+"/decision", `{"decision":"approved"}`)
	mustStatus(t, rec, http.StatusForbidden)
}

// TestDecide_InvalidInput_FR007 proves a decision value outside approved/rejected
// is rejected as a validation error.
func TestDecide_InvalidInput_FR007(t *testing.T) {
	h := newHarness(t, "admin_decide_invalid")
	reqID := submitAsBuyer(t, h, "3273010101010001")

	asAdmin(t, h, "admin@contoh.test")
	rec := h.doJSON("POST", "/api/admin/verification/"+reqID+"/decision", `{"decision":"maybe"}`)
	mustStatus(t, rec, http.StatusUnprocessableEntity)
	if p := decodeProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %s", p.Code, httpx.CodeValidationFailed)
	}
}
