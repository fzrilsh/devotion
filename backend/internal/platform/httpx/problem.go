package httpx

import (
	"encoding/json"
	"net/http"
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

func writeProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// WriteInternal writes a generic 500 problem+json. A panic is an internal fault:
// the client learns nothing beyond that something failed, and the stack goes to
// the log via the Recover middleware.
func WriteInternal(w http.ResponseWriter) {
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
