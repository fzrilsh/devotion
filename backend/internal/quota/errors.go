package quota

import (
	"errors"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// Sentinel errors the service layer returns and http.go translates to problem
// codes. They are package-private and lowercase, matching the listing package.
var (
	// errProfileMissing means the buyer's account has no business profile. A
	// profile has existed since registration, so this is an invariant break
	// surfaced as a 500 rather than a 404 the buyer can act on.
	errProfileMissing = errors.New("quota: profil tidak ditemukan")
)

// conflictError carries the code and the already-composed Indonesian detail a
// 409 must quote. The self-request rejection (FR-083) is the only conflict this
// package raises, but the shape mirrors listing so writeErr stays uniform.
type conflictError struct {
	code   httpx.Code
	detail string
}

func (e *conflictError) Error() string { return e.detail }

// validationError lets the service reject bad input with the same per-field
// shape httpx.WriteValidation renders, so a handler translates it without
// re-stating the field rules. Unknown or unpublished listing ids (422) come
// back this way too, since the buyer resolves them by editing the request.
type validationError struct {
	fields []httpx.FieldError
}

func (e *validationError) Error() string { return "quota: masukan tidak sah" }
