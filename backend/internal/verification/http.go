// Package verification wires the applicant-facing HTTP handlers for identity
// verification: uploading the two required files, submitting a verification
// request, and reading one's own submission history. The heavy lifting (magic
// byte checks, EXIF stripping, quota, owner-or-admin access) lives in the
// storage package; this package is the thin HTTP layer over it plus the
// verification_request rows.
//
// All four routes are gated to the two business roles. Admin has no
// business_profile, so an admin caller is refused at the router before reaching
// a handler that would otherwise resolve a non-existent profile and 500.
package verification

import (
	"errors"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
	"github.com/fzrilsh/devotion/backend/internal/platform/storage"
)

// Service owns the verification and file endpoints. It holds the pool for the
// verification_request rows, the clock for the submission timestamp, and the
// storage service that validates and stores the files.
type Service struct {
	pool    *pgxpool.Pool
	clock   platform.Clock
	storage *storage.Service
}

// New builds the Service. The storage service is constructed once at startup and
// shared, so it is passed in rather than built here.
func New(pool *pgxpool.Pool, clock platform.Clock, store *storage.Service) *Service {
	return &Service{pool: pool, clock: clock, storage: store}
}

func (s *Service) queries() *sqlcgen.Queries { return sqlcgen.New(s.pool) }

// Register mounts the four applicant routes. Three of them (uploading a file,
// submitting a request, reading one's own submissions) are applicant actions
// gated to the two business roles: an admin has no business_profile to attach a
// file or a submission to, so admin is refused at the router before a handler
// would resolve a non-existent profile and 500.
//
// GET /api/files/{fileId} also admits admin. SC-012 requires that a stored
// identity document or location photo be reachable by its owner and by an admin,
// and no one else. Without admin here, the Fase 7 verification queue could never
// open a submitted document. The owner-or-admin decision itself lives in
// storage.Open; this gate only widens the door to the two roles that may knock.
// Literal paths are registered before the {fileId} pattern so the router matches
// them first.
func (s *Service) Register(r *httpx.Router, auth httpx.Authenticator) {
	business := httpx.RequireRole(auth, httpx.RoleSubcontractor, httpx.RoleBuyer)
	businessOrAdmin := httpx.RequireRole(auth, httpx.RoleSubcontractor, httpx.RoleBuyer, httpx.RoleAdmin)

	r.Gated("POST /api/files", business, s.uploadFile)
	r.Gated("GET /api/files/{fileId}", businessOrAdmin, s.getFile)
	r.Gated("POST /api/verification", business, s.submit)
	r.Gated("GET /api/verification", business, s.list)
}

// resolveProfile maps the authenticated account to its business_profile id. A
// gated business caller always has one; a missing row is an invariant break
// (the gate should have refused an admin), so it writes a 500 and returns false.
func (s *Service) resolveProfile(w http.ResponseWriter, r *http.Request, accountID pgtype.UUID) (pgtype.UUID, bool) {
	id, err := s.queries().GetProfileIDByAccount(r.Context(), accountID)
	if err != nil {
		httpx.WriteInternal(w)
		return pgtype.UUID{}, false
	}
	return id, true
}

// uploadResponse is the contract's UploadedFile: the created file's id, its
// declared kind, the verified content type, and the stored size in bytes.
type uploadResponse struct {
	FileID      string `json:"file_id"`
	Kind        string `json:"kind"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

// uploadFile handles POST /api/files: one multipart part named "file" plus a
// "kind" field. The body is bounded to the per-file limit before parsing so an
// oversized upload never spills to the box's disk. storage.Save does the real
// validation (magic bytes, EXIF, quota); this maps its sentinels to problems.
func (s *Service) uploadFile(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	owner, ok := s.resolveProfile(w, r, acc.ID)
	if !ok {
		return
	}

	// Bound the whole request body before ParseMultipartForm reads it. The slack
	// covers multipart framing (boundaries, headers) around the file itself.
	const framingSlack = 1 << 20
	r.Body = http.MaxBytesReader(w, r.Body, s.storage.MaxFileBytes()+framingSlack)

	// Keep the in-memory portion small; the reader handed to Save streams the
	// rest. 1MB in memory is well under the box's budget.
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		httpx.WriteProblem(w, httpx.CodeFileTooLarge, "Berkas melebihi batas ukuran.")
		return
	}

	fileType, valid := parseKind(r.FormValue("kind"))
	if !valid {
		httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
			{Field: "kind", Message: "Jenis berkas harus identity_document atau location_photo."},
		})
		return
	}

	part, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
			{Field: "file", Message: "Berkas wajib diunggah."},
		})
		return
	}
	defer part.Close()

	saved, err := s.storage.Save(r.Context(), owner, fileType, header.Filename, part)
	if err != nil {
		writeStorageErr(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, uploadResponse{
		FileID:      uuidString(saved.File.ID),
		Kind:        string(saved.File.Type),
		ContentType: saved.File.MimeType,
		SizeBytes:   int64(saved.File.SizeBytes),
	})
}

// getFile handles GET /api/files/{fileId}: it streams a stored file's bytes to
// its owner or to an admin, and refuses everyone else with 403. Access control
// is the only thing guarding an identity document, so the owner-or-admin check
// in storage.Open is the whole point of this route (FR-009).
func (s *Service) getFile(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}

	id, valid := parseUUID(r.PathValue("fileId"))
	if !valid {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Id berkas tidak sah.")
		return
	}

	caller := storage.Caller{IsAdmin: callerIsAdmin(r)}
	// A non-admin caller is matched by profile. An admin needs no profile and
	// passes the owner check unconditionally, so resolve the profile only when
	// the caller is not an admin (an admin has none to resolve).
	if !caller.IsAdmin {
		profileID, ok := s.resolveProfile(w, r, acc.ID)
		if !ok {
			return
		}
		caller.ProfileID = profileID
	}

	body, row, err := s.storage.Open(r.Context(), id, caller)
	if err != nil {
		writeStorageErr(w, err)
		return
	}
	defer body.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "inline; filename=\""+row.StoragePath+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, body)
}

// parseKind maps the wire enum to the sqlc FileType, reporting validity. Any
// value outside the two allowed kinds is rejected before touching storage.
func parseKind(v string) (sqlcgen.FileType, bool) {
	switch v {
	case string(sqlcgen.FileTypeIdentityDocument):
		return sqlcgen.FileTypeIdentityDocument, true
	case string(sqlcgen.FileTypeLocationPhoto):
		return sqlcgen.FileTypeLocationPhoto, true
	default:
		return "", false
	}
}

// writeStorageErr maps a storage sentinel to its problem response. An unknown
// error is a 500; the sentinels cover every rejection storage.Save and
// storage.Open can produce.
func writeStorageErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrTooLarge):
		httpx.WriteProblem(w, httpx.CodeFileTooLarge, "Berkas melebihi batas ukuran.")
	case errors.Is(err, storage.ErrUnsupportedType):
		httpx.WriteProblem(w, httpx.CodeUnsupportedFileType, "Tipe berkas tidak didukung. Unggah JPEG, PNG, atau PDF.")
	case errors.Is(err, storage.ErrQuotaFull):
		httpx.WriteProblem(w, httpx.CodeStorageQuotaFull, "Kuota penyimpanan penuh.")
	case errors.Is(err, storage.ErrForbidden):
		httpx.WriteProblem(w, httpx.CodeForbidden, "Anda tidak berhak mengakses berkas ini.")
	case errors.Is(err, storage.ErrNotFound):
		httpx.WriteProblem(w, httpx.CodeNotFound, "Berkas tidak ditemukan.")
	default:
		httpx.WriteInternal(w)
	}
}
