package verification

import (
	"net/http"
	"testing"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// TestUploadFile_Success_FR006 proves a business caller can upload one of the
// two required files and receive its id, verified content type, and size.
func TestUploadFile_Success_FR006(t *testing.T) {
	h := newHarness(t, "verif_upload_ok")

	rec := h.uploadMultipart(t, "identity_document", jpegBytes(t))
	mustStatus(t, rec, http.StatusCreated)

	got := decodeUpload(t, rec)
	if got.FileID == "" {
		t.Fatal("file_id kosong")
	}
	if got.Kind != "identity_document" {
		t.Fatalf("kind %q, mau identity_document", got.Kind)
	}
	if got.ContentType != "image/jpeg" {
		t.Fatalf("content_type %q, mau image/jpeg", got.ContentType)
	}
	if got.SizeBytes <= 0 {
		t.Fatalf("size_bytes %d, mau > 0", got.SizeBytes)
	}
}

// TestUploadFile_AdminRejected_FR009 proves an admin, who holds neither business
// role, is refused at the gate with 403 before reaching the handler. Admins have
// no business_profile to own a file.
func TestUploadFile_AdminRejected_FR009(t *testing.T) {
	h := newHarness(t, "verif_upload_admin")
	adminAcc := seedAdmin(t, h.pool, "admin@contoh.test")
	h.asPrincipal(&httpx.Principal{
		Roles:   httpx.RoleAdmin,
		Account: sqlcgen.UserAccount{ID: adminAcc},
	})

	rec := h.uploadMultipart(t, "identity_document", jpegBytes(t))
	mustStatus(t, rec, http.StatusForbidden)
	if p := decodeProblem(t, rec); p.Code != string(httpx.CodeForbidden) {
		t.Fatalf("code %q, mau %s", p.Code, httpx.CodeForbidden)
	}
}

// TestUploadFile_Unauthenticated_FR006 proves an anonymous caller is refused
// with 401 before any file work happens.
func TestUploadFile_Unauthenticated_FR006(t *testing.T) {
	h := newHarness(t, "verif_upload_anon")
	h.asPrincipal(nil)

	rec := h.uploadMultipart(t, "identity_document", jpegBytes(t))
	mustStatus(t, rec, http.StatusUnauthorized)
}

// TestUploadFile_InvalidKind_FR006 proves an unknown kind is rejected as a
// validation error before storage is touched.
func TestUploadFile_InvalidKind_FR006(t *testing.T) {
	h := newHarness(t, "verif_upload_kind")

	rec := h.uploadMultipart(t, "passport", jpegBytes(t))
	mustStatus(t, rec, http.StatusUnprocessableEntity)
	if p := decodeProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %s", p.Code, httpx.CodeValidationFailed)
	}
}

// TestUploadFile_UnsupportedType_FR006 proves a body whose real bytes are not an
// allowed image or PDF is refused with 415, regardless of the .jpg filename.
func TestUploadFile_UnsupportedType_FR006(t *testing.T) {
	h := newHarness(t, "verif_upload_type")

	rec := h.uploadMultipart(t, "identity_document", []byte("bukan gambar, hanya teks biasa"))
	mustStatus(t, rec, http.StatusUnsupportedMediaType)
	if p := decodeProblem(t, rec); p.Code != string(httpx.CodeUnsupportedFileType) {
		t.Fatalf("code %q, mau %s", p.Code, httpx.CodeUnsupportedFileType)
	}
}

// TestGetFile_Owner_FR009 proves the file's owner can stream its bytes back.
func TestGetFile_Owner_FR009(t *testing.T) {
	h := newHarness(t, "verif_getfile_owner")

	up := decodeUpload(t, h.uploadMultipart(t, "identity_document", jpegBytes(t)))
	rec := h.do("GET", "/api/files/"+up.FileID)
	mustStatus(t, rec, http.StatusOK)
	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("content-type %q, mau application/octet-stream", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("body kosong")
	}
}

// TestGetFile_StrangerForbidden_FR009 is the load-bearing case: a business
// caller who is neither the file's owner nor an admin is refused with 403.
// Access control is the only guard on an identity document.
func TestGetFile_StrangerForbidden_FR009(t *testing.T) {
	h := newHarness(t, "verif_getfile_stranger")

	// Owner uploads.
	up := decodeUpload(t, h.uploadMultipart(t, "identity_document", jpegBytes(t)))

	// A different business account (the stranger) tries to read it.
	strangerAcc, _ := seedProfile(t, h.pool, "stranger@contoh.test", true)
	h.asPrincipal(&httpx.Principal{
		Roles:   httpx.RoleSubcontractor,
		Account: sqlcgen.UserAccount{ID: strangerAcc},
	})

	rec := h.do("GET", "/api/files/"+up.FileID)
	mustStatus(t, rec, http.StatusForbidden)
	if p := decodeProblem(t, rec); p.Code != string(httpx.CodeForbidden) {
		t.Fatalf("code %q, mau %s", p.Code, httpx.CodeForbidden)
	}
}

// TestGetFile_AdminReads_SC012 is the positive half of SC-012: an admin, who
// owns no business_profile, can still stream a submitted identity document. The
// Fase 7 verification queue depends on this; without it an admin could never
// open a document to decide on it. The negative half is
// TestGetFile_StrangerForbidden_FR009.
func TestGetFile_AdminReads_SC012(t *testing.T) {
	h := newHarness(t, "verif_getfile_admin")

	// Owner (the harness buyer) uploads a document.
	up := decodeUpload(t, h.uploadMultipart(t, "identity_document", jpegBytes(t)))

	// An admin, holding no business role, reads it back.
	adminAcc := seedAdmin(t, h.pool, "admin@contoh.test")
	h.asPrincipal(&httpx.Principal{
		Roles:   httpx.RoleAdmin,
		Account: sqlcgen.UserAccount{ID: adminAcc},
	})

	rec := h.do("GET", "/api/files/"+up.FileID)
	mustStatus(t, rec, http.StatusOK)
	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("content-type %q, mau application/octet-stream", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("body kosong, admin harus dapat byte berkas")
	}
}

// TestGetFile_MalformedID_FR009 proves a non-UUID path segment is a validation
// error, not a 404, so a typo is distinguishable from a missing file.
func TestGetFile_MalformedID_FR009(t *testing.T) {
	h := newHarness(t, "verif_getfile_badid")

	rec := h.do("GET", "/api/files/bukan-uuid")
	mustStatus(t, rec, http.StatusUnprocessableEntity)
	if p := decodeProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %s", p.Code, httpx.CodeValidationFailed)
	}
}

// TestGetFile_UnknownID_FR009 proves a well-formed but unused id is a 404, and
// existence is not leaked to a caller who does not own it.
func TestGetFile_UnknownID_FR009(t *testing.T) {
	h := newHarness(t, "verif_getfile_unknown")

	rec := h.do("GET", "/api/files/ffffffff-ffff-ffff-ffff-ffffffffffff")
	mustStatus(t, rec, http.StatusNotFound)
}
