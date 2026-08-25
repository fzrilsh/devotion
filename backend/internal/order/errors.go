package order

import (
	"errors"
	"net/http"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// conflictError carries a stable machine code and the already-composed
// Indonesian detail a 4xx must quote. The accept path raises several: NOT_FOUND,
// FORBIDDEN, REQUEST_ALREADY_AGREED, CAPACITY_ALREADY_TAKEN,
// READINESS_AFTER_DEADLINE. Its shape mirrors quota so writeErr stays uniform.
type conflictError struct {
	code   httpx.Code
	detail string
}

func (e *conflictError) Error() string { return e.detail }

// metaError is a conflict-style rejection that also carries structured context
// under problem "meta". FR-035 uses it so the INSUFFICIENT_CAPACITY body states
// the remaining capacity and the deadline week both as a quotable Indonesian
// detail and as machine fields the client can render without parsing the string.
type metaError struct {
	code   httpx.Code
	detail string
	meta   map[string]any
}

func (e *metaError) Error() string { return e.detail }

// writeErr maps a service error to its problem response. A conflictError carries
// its own code and detail; a metaError carries structured context too; anything
// else is an unexpected fault surfaced as a 500.
func writeErr(w http.ResponseWriter, err error) {
	var cerr *conflictError
	if errors.As(err, &cerr) {
		httpx.WriteProblem(w, cerr.code, cerr.detail)
		return
	}
	var merr *metaError
	if errors.As(err, &merr) {
		httpx.WriteProblemMeta(w, merr.code, merr.detail, merr.meta)
		return
	}
	httpx.WriteInternal(w)
}
