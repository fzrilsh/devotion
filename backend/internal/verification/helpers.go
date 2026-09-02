package verification

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// maxBodyBytes caps a JSON request body. The verification payload is three
// short fields; this stops a client from streaming an unbounded body into the
// decoder. File uploads are bounded separately by the storage per-file limit.
const maxBodyBytes = 64 << 10

// writeJSON encodes v as the 2xx body. Error bodies go through
// httpx.WriteProblem; this is only for the success payloads the handlers own.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// decodeJSON reads a JSON body into dst, rejecting unknown fields and oversized
// bodies. It returns false and writes a validation problem on failure, so a
// handler can `if !decodeJSON(...) { return }`.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Format permintaan tidak sah.")
		return false
	}
	return true
}

// uuidString renders a pgtype.UUID as canonical text, empty when not valid.
func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b, _ := u.MarshalJSON()
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return ""
}

// parseUUID parses canonical UUID text into a pgtype.UUID, reporting validity.
func parseUUID(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, false
	}
	return u, u.Valid
}

// isNoRows reports whether err is pgx.ErrNoRows.
func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// isPendingViolation reports whether err is the partial unique index rejecting a
// second pending verification for one profile. The service turns it into a 409
// VERIFICATION_PENDING; a re-submission is allowed only after a rejection
// clears the pending row (FR-011).
func isPendingViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == "idx_one_pending_verification"
}

// principalAccount pulls the authenticated UserAccount off the request context.
// The route is gated, so a missing Principal or a wrong Account type is an
// invariant violation and becomes a 500. The bool is false when it already wrote
// the 500, so the handler returns early.
func principalAccount(w http.ResponseWriter, r *http.Request) (sqlcgen.UserAccount, bool) {
	p, ok := httpx.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteInternal(w)
		return sqlcgen.UserAccount{}, false
	}
	acc, ok := p.Account.(sqlcgen.UserAccount)
	if !ok {
		httpx.WriteInternal(w)
		return sqlcgen.UserAccount{}, false
	}
	return acc, true
}

// callerIsAdmin reports whether the gated principal holds the admin role. GET
// /files/{fileId} passes it into storage.Caller so an admin reads any file
// while a business caller is confined to their own (FR-009). The business gate
// keeps admins off these applicant routes, so in practice this is false here;
// it is threaded through anyway so the owner-or-admin rule lives in one place.
func callerIsAdmin(r *http.Request) bool {
	p, ok := httpx.PrincipalFromContext(r.Context())
	if !ok {
		return false
	}
	return p.Roles.Has(httpx.RoleAdmin)
}

// sourceAddr parses the request's true origin (already rewritten by RealIP) into
// the shape CreateVerificationRequest records for audit. An unparsable address
// yields nil, which stores NULL.
func sourceAddr(r *http.Request) *netip.Addr {
	host := r.RemoteAddr
	if ap, err := netip.ParseAddrPort(host); err == nil {
		a := ap.Addr()
		return &a
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return &addr
	}
	return nil
}

// strPtr returns a pointer to s, used so a required-but-empty string still
// serializes as a JSON string rather than being confused with null.
func strPtr(s string) *string { return &s }

// uuidPtr renders a pgtype.UUID as a canonical-text pointer, nil when the id is
// not valid so the field serializes as JSON null.
func uuidPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuidString(u)
	return &s
}

// textPtr renders a nullable pgtype.Text as a pointer, nil when not valid. It
// backs the contract's reason field (admin_note), which is null until an admin
// decides.
func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// tstzPtr renders a nullable timestamptz as an RFC 3339 string pointer, nil when
// not valid. decided_at is null on a pending row; created_at is always set.
func tstzPtr(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(time.RFC3339)
	return &s
}

