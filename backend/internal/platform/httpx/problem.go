package httpx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fzrilsh/devotion/backend/internal/platform/observability"
)

// problemContentType is the RFC 9457 media type. Every error body uses it so a
// mistyped endpoint returns problem+json, never HTML.
const problemContentType = "application/problem+json"

// typeBaseURI prefixes the machine-readable type URI. openapi.yaml uses
// https://devotion/errors/<slug>; the slug is derived from the code.
const typeBaseURI = "https://devotion/errors/"

// FieldError is one field-level validation message, matching the errors[] entry
// in openapi.yaml's ProblemValidation.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Problem is the RFC 9457 error body. Errors carries the per-field list for a
// validation failure (ProblemValidation in the contract); it is omitted for
// every other code. Meta carries code-specific structured context.
type Problem struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Code     Code           `json:"code"`
	Detail   string         `json:"detail"`
	Instance string         `json:"instance,omitempty"`
	Errors   []FieldError   `json:"errors,omitempty"`
	Meta     map[string]any `json:"meta,omitempty"`
}

// WriteProblem writes code's status, title, and the given Indonesian detail as
// problem+json. The status and title come from the code, so callers cannot pair
// a code with an inconsistent status.
func WriteProblem(w http.ResponseWriter, code Code, detail string) {
	writeProblem(w, Problem{
		Type:   typeBaseURI + slugFor(code),
		Title:  TitleFor(code),
		Status: StatusFor(code),
		Code:   code,
		Detail: detail,
	})
}

// WriteValidation writes a VALIDATION_FAILED problem carrying the field errors.
func WriteValidation(w http.ResponseWriter, detail string, fields []FieldError) {
	writeProblem(w, Problem{
		Type:   typeBaseURI + slugFor(CodeValidationFailed),
		Title:  TitleFor(CodeValidationFailed),
		Status: StatusFor(CodeValidationFailed),
		Code:   CodeValidationFailed,
		Detail: detail,
		Errors: fields,
	})
}

// WriteProblemMeta writes a problem+json like WriteProblem but attaches the
// given structured context under "meta". FR-035 uses it to state the actual
// remaining capacity and the until-week the reply cannot cover, so a client can
// render the shortfall without parsing the Indonesian detail string.
func WriteProblemMeta(w http.ResponseWriter, code Code, detail string, meta map[string]any) {
	writeProblem(w, Problem{
		Type:   typeBaseURI + slugFor(code),
		Title:  TitleFor(code),
		Status: StatusFor(code),
		Code:   code,
		Detail: detail,
		Meta:   meta,
	})
}

func writeProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// WriteInternal writes a generic 500 problem+json and reports the internal
// failure to Sentry. When the caller has the original error, it is captured as
// an exception. A no-argument call remains valid for invariant failures where
// there is no safe error value to attach.
func WriteInternal(w http.ResponseWriter, cause ...error) {
	requestID := RequestIDFromContext(requestContextFromWriter(w))
	if len(cause) > 0 && cause[0] != nil {
		observability.CaptureException(cause[0], requestID)
	} else {
		observability.CaptureMessage("internal server error", requestID)
	}
	writeInternalResponse(w)
}

// requestContextWriter is implemented by the request logger's response writer.
// It lets generic response helpers correlate captured failures with request logs
// without changing every handler signature.
type requestContextWriter interface {
	RequestContext() context.Context
}

func requestContextFromWriter(w http.ResponseWriter) context.Context {
	if rw, ok := w.(requestContextWriter); ok {
		return rw.RequestContext()
	}
	return context.Background()
}

func writeInternalResponse(w http.ResponseWriter) {
	writeProblem(w, Problem{
		Type:   typeBaseURI + "internal",
		Title:  "Terjadi galat",
		Status: http.StatusInternalServerError,
		Code:   Code("INTERNAL_ERROR"),
		Detail: "Terjadi galat internal. Silakan coba lagi.",
	})
}

// slugFor turns a code into the kebab-case slug used in the type URI. It lower-
// cases and replaces underscores with hyphens: VALIDATION_FAILED ->
// validation-failed.
func slugFor(code Code) string {
	b := make([]byte, 0, len(code))
	for i := 0; i < len(code); i++ {
		c := code[i]
		switch {
		case c == '_':
			b = append(b, '-')
		case c >= 'A' && c <= 'Z':
			b = append(b, c+('a'-'A'))
		default:
			b = append(b, c)
		}
	}
	return string(b)
}
