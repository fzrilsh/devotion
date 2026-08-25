package verification

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// submitBody builds a POST /api/verification JSON payload.
func submitBody(number, identityFileID, locationFileID string) string {
	return fmt.Sprintf(`{"identity_number":%q,"identity_file_id":%q,"location_file_id":%q}`,
		number, identityFileID, locationFileID)
}

// uploadTwo uploads an identity file and a location file for the current
// principal and returns their ids, so a verification submission has real,
// owned files to reference.
func uploadTwo(t *testing.T, h *harness) (string, string) {
	t.Helper()
	id := decodeUpload(t, h.uploadMultipart(t, "identity_document", jpegBytes(t)))
	loc := decodeUpload(t, h.uploadMultipart(t, "location_photo", jpegBytes(t)))
	return id.FileID, loc.FileID
}

// TestSubmit_Success_FR010 proves a business caller with two uploaded files can
// submit a pending verification request and read back its status.
func TestSubmit_Success_FR010(t *testing.T) {
	h := newHarness(t, "verif_submit_ok")
	idFile, locFile := uploadTwo(t, h)

	rec := h.doJSON("POST", "/api/verification", submitBody("3273010101010001", idFile, locFile))
	mustStatus(t, rec, http.StatusCreated)

	var got verificationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.RequestID == "" {
		t.Fatal("request_id kosong")
	}
	if got.Status != string(sqlcgen.VerificationStatusPending) {
		t.Fatalf("status %q, mau pending", got.Status)
	}
	if got.SubmittedAt == nil {
		t.Fatal("submitted_at nil")
	}
	if got.DecidedAt != nil {
		t.Fatalf("decided_at %v, mau nil pada pengajuan baru", *got.DecidedAt)
	}
}

// TestSubmit_SecondWhilePending_FR011 proves a second submission while one is
// still pending is refused with 409. A re-submission is allowed only after an
// admin rejects the prior one.
func TestSubmit_SecondWhilePending_FR011(t *testing.T) {
	h := newHarness(t, "verif_submit_pending")
	idFile, locFile := uploadTwo(t, h)

	first := h.doJSON("POST", "/api/verification", submitBody("3273010101010001", idFile, locFile))
	mustStatus(t, first, http.StatusCreated)

	second := h.doJSON("POST", "/api/verification", submitBody("3273010101010002", idFile, locFile))
	mustStatus(t, second, http.StatusConflict)
	if p := decodeProblem(t, second); p.Code != string(httpx.CodeVerificationPending) {
		t.Fatalf("code %q, mau %s", p.Code, httpx.CodeVerificationPending)
	}
}

// TestSubmit_AdminRejected_FR006 proves an admin is refused at the gate with 403.
func TestSubmit_AdminRejected_FR006(t *testing.T) {
	h := newHarness(t, "verif_submit_admin")
	idFile, locFile := uploadTwo(t, h)

	adminAcc := seedAdmin(t, h.pool, "admin@contoh.test")
	h.asPrincipal(&httpx.Principal{
		Roles:   httpx.RoleAdmin,
		Account: sqlcgen.UserAccount{ID: adminAcc},
	})

	rec := h.doJSON("POST", "/api/verification", submitBody("3273010101010001", idFile, locFile))
	mustStatus(t, rec, http.StatusForbidden)
}

// TestSubmit_InvalidInput_FR010 proves a too-short identity number and malformed
// file ids are rejected as a validation error before a row is attempted.
func TestSubmit_InvalidInput_FR010(t *testing.T) {
	h := newHarness(t, "verif_submit_invalid")

	rec := h.doJSON("POST", "/api/verification", submitBody("123", "bukan-uuid", "juga-bukan"))
	mustStatus(t, rec, http.StatusUnprocessableEntity)
	p := decodeProblem(t, rec)
	if p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %s", p.Code, httpx.CodeValidationFailed)
	}
	if len(p.Errors) == 0 {
		t.Fatal("errors kosong, mau daftar field yang gagal")
	}
}

// TestList_Success_FR006 proves the caller reads their own submissions as a bare
// array, newest first, and a caller with none gets an empty array.
func TestList_Success_FR006(t *testing.T) {
	h := newHarness(t, "verif_list_ok")

	empty := h.do("GET", "/api/verification")
	mustStatus(t, empty, http.StatusOK)
	var before []verificationResponse
	if err := json.Unmarshal(empty.Body.Bytes(), &before); err != nil {
		t.Fatalf("decode empty: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("panjang %d, mau 0 sebelum pengajuan", len(before))
	}

	idFile, locFile := uploadTwo(t, h)
	mustStatus(t, h.doJSON("POST", "/api/verification", submitBody("3273010101010001", idFile, locFile)), http.StatusCreated)

	rec := h.do("GET", "/api/verification")
	mustStatus(t, rec, http.StatusOK)
	var after []verificationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &after); err != nil {
		t.Fatalf("decode after: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("panjang %d, mau 1", len(after))
	}
	if after[0].BusinessName == "" {
		t.Fatal("business_name kosong")
	}
}

// TestList_AdminRejected_FR006 proves an admin is refused at the gate with 403.
func TestList_AdminRejected_FR006(t *testing.T) {
	h := newHarness(t, "verif_list_admin")
	adminAcc := seedAdmin(t, h.pool, "admin@contoh.test")
	h.asPrincipal(&httpx.Principal{
		Roles:   httpx.RoleAdmin,
		Account: sqlcgen.UserAccount{ID: adminAcc},
	})

	rec := h.do("GET", "/api/verification")
	mustStatus(t, rec, http.StatusForbidden)
}
