package listing

import (
	"errors"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// Sentinel errors the service layer returns and http.go translates to problem
// codes. They are package-private and lowercase, matching the account package.
var (
	errListingNotFound = errors.New("listing: listing tidak ditemukan")
	errListingExists   = errors.New("listing: profil sudah punya listing")
	errProfileMissing  = errors.New("listing: profil tidak ditemukan")
)

// conflictError carries the code and the already-composed Indonesian detail a
// 409 must quote. Two different sentences share CAPACITY_ALREADY_ALLOCATED (one
// on PUT /listing/me, another on PUT /listing/me/periods) and PERIOD_ALREADY_
// ALLOCATED has its own, so the caller composes the exact wording with
// platform.FormatDateID and hands it here rather than assembling it in one
// place that cannot know which sentence applies.
type conflictError struct {
	code   httpx.Code
	detail string
	// week and used are kept for tests and future callers that want the raw
	// figures; the wire response reads only code and detail.
	week time.Time
	used int32
	want int32
}

func (e *conflictError) Error() string { return e.detail }

// validationError lets the service reject bad input with the same per-field
// shape httpx.WriteValidation renders, so a handler translates it without
// re-stating the field rules.
type validationError struct {
	fields []httpx.FieldError
}

func (e *validationError) Error() string { return "listing: masukan tidak sah" }
