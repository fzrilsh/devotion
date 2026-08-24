package search

import "github.com/fzrilsh/devotion/backend/internal/platform/httpx"

// validationError lets the handler reject bad query parameters with the same
// per-field shape httpx.WriteValidation renders. Search validates its inputs in
// the handler, so this stays small: one type carrying the field list.
type validationError struct {
	fields []httpx.FieldError
}

func (e *validationError) Error() string { return "search: masukan tidak sah" }
