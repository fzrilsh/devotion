package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/platform/cloudflare"
	"github.com/fzrilsh/devotion/backend/internal/platform/observability"
)

// Middleware wraps an http.Handler with cross-cutting behavior.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware so that the first listed runs outermost. Chain(a, b)
// produces a(b(h)): a sees the request first and the response last.
func Chain(mw ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for i := len(mw) - 1; i >= 0; i-- {
			h = mw[i](h)
		}
		return h
	}
}

// requestIDHeader is the response header echoing the per-request ID so it can be
// correlated with logs during manual testing.
const requestIDHeader = "X-Request-ID"

// RequestID assigns each request a random ID, stores it on the context, and
// echoes it back. An inbound X-Request-ID is honored so an edge proxy can thread
// its own correlation ID through.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(WithRequestID(r.Context(), id)))
	})
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// rand.Read never fails on the platforms we run on; fall back to a
		// fixed marker rather than panicking a request over a correlation ID.
		return "req-fallback"
	}
	return hex.EncodeToString(b[:])
}

// Recover turns a panic into a 500 problem+json. The stack goes to slog, never
// to the client: a leaked stack is an information disclosure.
func Recover(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					stack := debug.Stack()
					log.ErrorContext(r.Context(), "panic dipulihkan",
						"panic", p, "stack", string(stack))
					observability.CapturePanic(p, RequestIDFromContext(r.Context()))
					writeInternalResponse(w)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// statusRecorder captures the status code written by a handler for the logger.
type statusRecorder struct {
	http.ResponseWriter
	ctx    context.Context
	status int
}

// RequestContext lets error helpers correlate a response with its request
// without changing every existing handler signature.
func (rec *statusRecorder) RequestContext() context.Context { return rec.ctx }

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

// Logger logs one JSON line per request with request_id, method, path, status,
// and duration_ms. The request ID rides the context via the contextHandler.
func Logger(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, ctx: r.Context(), status: http.StatusOK}
			next.ServeHTTP(rec, r)
			log.InfoContext(r.Context(), "request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

// RealIP replaces RemoteAddr with the client address derived from the Cloudflare
// trust boundary, so downstream rate limiting and audit logs see the true origin
// rather than the edge proxy. It never trusts a forwarded header from a
// connection outside the pinned Cloudflare ranges.
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := cloudflare.RealIP(r); ip != "" {
			r.RemoteAddr = ip
		}
		next.ServeHTTP(w, r)
	})
}
