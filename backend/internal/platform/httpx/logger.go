package httpx

import (
	"context"
	"log/slog"
	"os"
)

// ctxKey is the private context key type for values httpx stores on the request
// context. A private type keeps other packages from colliding with these keys.
type ctxKey int

const (
	// requestIDKey holds the per-request ID set by the RequestID middleware.
	requestIDKey ctxKey = iota
)

// WithRequestID returns a copy of ctx carrying the request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext returns the request ID stored on ctx, or "" if none.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// contextHandler wraps a slog.Handler and pulls the request ID off the context
// onto every record. Doing this here, once, is why no log call site can forget
// to attach it.
type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestIDFromContext(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

// NewLogger returns a JSON slog.Logger whose records carry request_id whenever
// it is present on the logging context.
func NewLogger() *slog.Logger {
	base := slog.NewJSONHandler(os.Stdout, nil)
	return slog.New(contextHandler{Handler: base})
}
