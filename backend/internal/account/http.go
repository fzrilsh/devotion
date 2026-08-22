package account

import (
	"encoding/json"
	"net/http"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// recoverFloor is the fixed duration handleRecoverRequest pads its reply to, so
// the response time does not leak whether the account existed. It must exceed a
// realistic lookup-plus-issue path; a missing account short-circuits and would
// otherwise reply faster.
const recoverFloor = 400 * time.Millisecond

// maxBodyBytes caps a request body. Auth payloads are tiny; this stops a client
// from streaming an unbounded body into the decoder.
const maxBodyBytes = 64 << 10

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

// emailRe is a pragmatic email check: something before an @, a domain with a
// dot after. The authoritative validation is that a code can be delivered; this
// only rejects obvious garbage before we touch the database.
var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// phoneRe matches an E.164 Indonesian number: +62 followed by 8 to 13 digits.
var phoneRe = regexp.MustCompile(`^\+62[0-9]{8,13}$`)

// codeRe matches the six-digit codes this service mints.
var codeRe = regexp.MustCompile(`^[0-9]{6}$`)

// normalizeEmail lowercases and trims an email so lookups are case-insensitive
// and stable against stray whitespace. The column is citext, but normalizing in
// Go keeps the stored value tidy too.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// normalizePhone trims whitespace and drops a leading "+" so the stored value
// matches the phone_format constraint (^62...), while the wire form the client
// sends stays E.164 with the plus. A number without a plus is left as-is.
func normalizePhone(phone string) string {
	return strings.TrimPrefix(strings.TrimSpace(phone), "+")
}

// clientAddr extracts the source IP from RemoteAddr (already rewritten by the
// RealIP middleware to the true origin) for the per-address OTP budget. A value
// that will not parse yields "", which the caller treats as no address.
func clientAddr(r *http.Request) string {
	host := r.RemoteAddr
	if ap, err := netip.ParseAddrPort(host); err == nil {
		return ap.Addr().String()
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.String()
	}
	// Fall back to stripping a trailing :port if present.
	if i := strings.LastIndex(host, ":"); i >= 0 {
		if addr, err := netip.ParseAddr(host[:i]); err == nil {
			return addr.String()
		}
	}
	return ""
}

// sourceAddr parses the request's true origin (already rewritten by RealIP) into
// the shape session.Issue records for audit. An unparsable address yields nil,
// which stores NULL.
func sourceAddr(r *http.Request) *netip.Addr {
	host := r.RemoteAddr
	if ap, err := netip.ParseAddrPort(host); err == nil {
		a := ap.Addr()
		return &a
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return &addr
	}
	if i := strings.LastIndex(host, ":"); i >= 0 {
		if addr, err := netip.ParseAddr(host[:i]); err == nil {
			return &addr
		}
	}
	return nil
}

// writeJSON encodes v as the success body at the given status. Errors bodies go
// through httpx.WriteProblem; this is only for 2xx payloads the handlers own.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
