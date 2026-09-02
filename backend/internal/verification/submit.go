package verification

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// submitRequest is the POST /api/verification body: the identity number plus the
// two file ids the applicant uploaded first. All three are required; the files
// must already exist as uploaded_file rows owned by the same profile (the FK
// enforces existence, ownership is the applicant's own upload).
type submitRequest struct {
	IdentityNumber string `json:"identity_number"`
	IdentityFileID string `json:"identity_file_id"`
	LocationFileID string `json:"location_file_id"`
}

// verificationResponse is the contract's VerificationRequest. Nullable fields
// use pointers so a pending submission serializes reason and decided_at as null
// rather than a zero value. submitted_at maps created_at; reason maps admin_note.
type verificationResponse struct {
	RequestID      string  `json:"request_id"`
	Status         string  `json:"status"`
	ProfileID      string  `json:"profile_id"`
	BusinessName   string  `json:"business_name"`
	IdentityNumber *string `json:"identity_number"`
	IdentityFileID *string `json:"identity_file_id"`
	LocationFileID *string `json:"location_file_id"`
	Reason         *string `json:"reason"`
	SubmittedAt    *string `json:"submitted_at"`
	DecidedAt      *string `json:"decided_at"`
}

// submit handles POST /api/verification: it records one pending verification
// request for the caller's profile. A second submission while one is still
// pending is refused with 409 (FR-011); the applicant may re-submit only after
// an admin rejects the prior one, which clears the pending row.
func (s *Service) submit(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}

	var body submitRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	if fields := validateSubmit(body); len(fields) > 0 {
		httpx.WriteValidation(w, "Masukan tidak sah.", fields)
		return
	}

	identityFile, _ := parseUUID(body.IdentityFileID)
	locationFile, _ := parseUUID(body.LocationFileID)

	profileID, ok := s.resolveProfile(w, r, acc.ID)
	if !ok {
		return
	}

	row, err := s.queries().CreateVerificationRequest(r.Context(), sqlcgen.CreateVerificationRequestParams{
		ProfileID:              profileID,
		IdentityNumber:         strings.TrimSpace(body.IdentityNumber),
		IdentityFileID:         identityFile,
		LocationFileID:         locationFile,
		ApplicantSourceAddress: sourceAddr(r),
		CreatedAt:              pgtype.Timestamptz{Time: s.clock.Now(), Valid: true},
	})
	if err != nil {
		if isPendingViolation(err) {
			httpx.WriteProblem(w, httpx.CodeVerificationPending, "Anda masih memiliki pengajuan verifikasi yang menunggu keputusan.")
			return
		}
		httpx.WriteInternal(w)
		return
	}

	writeJSON(w, http.StatusCreated, verificationResponse{
		RequestID:      uuidString(row.ID),
		Status:         string(row.Status),
		ProfileID:      uuidString(row.ProfileID),
		IdentityNumber: strPtr(row.IdentityNumber),
		IdentityFileID: uuidPtr(row.IdentityFileID),
		LocationFileID: uuidPtr(row.LocationFileID),
		Reason:         textPtr(row.AdminNote),
		SubmittedAt:    tstzPtr(row.CreatedAt),
		DecidedAt:      tstzPtr(row.DecidedAt),
	})
}

// list handles GET /api/verification: it returns every submission the caller's
// profile has made, newest first, as a bare JSON array (FR-006). A caller with
// no submissions gets an empty array, not 404.
func (s *Service) list(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	profileID, ok := s.resolveProfile(w, r, acc.ID)
	if !ok {
		return
	}

	rows, err := s.queries().ListVerificationRequestsByProfile(r.Context(), profileID)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}

	out := make([]verificationResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, verificationResponse{
			RequestID:      uuidString(row.ID),
			Status:         string(row.Status),
			ProfileID:      uuidString(row.ProfileID),
			BusinessName:   row.BusinessName,
			IdentityNumber: strPtr(row.IdentityNumber),
			IdentityFileID: uuidPtr(row.IdentityFileID),
			LocationFileID: uuidPtr(row.LocationFileID),
			Reason:         textPtr(row.AdminNote),
			SubmittedAt:    tstzPtr(row.CreatedAt),
			DecidedAt:      tstzPtr(row.DecidedAt),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// validateSubmit enforces the contract's field rules before a row is attempted:
// identity_number is 8 to 32 characters after trimming, and both file ids are
// well-formed UUIDs. It returns one FieldError per broken rule.
func validateSubmit(body submitRequest) []httpx.FieldError {
	var fields []httpx.FieldError

	num := strings.TrimSpace(body.IdentityNumber)
	if n := len(num); n < 8 || n > 32 {
		fields = append(fields, httpx.FieldError{
			Field:   "identity_number",
			Message: "Nomor identitas harus 8 sampai 32 karakter.",
		})
	}
	if _, ok := parseUUID(body.IdentityFileID); !ok {
		fields = append(fields, httpx.FieldError{
			Field:   "identity_file_id",
			Message: "Id berkas identitas tidak sah.",
		})
	}
	if _, ok := parseUUID(body.LocationFileID); !ok {
		fields = append(fields, httpx.FieldError{
			Field:   "location_file_id",
			Message: "Id berkas lokasi tidak sah.",
		})
	}
	return fields
}
