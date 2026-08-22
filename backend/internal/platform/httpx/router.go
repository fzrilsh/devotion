package httpx

import (
	"log/slog"
	"net/http"
)

// apiPrefix is the single mount point for the JSON API. Everything under it is
// JSON; a catch-all returns 404 problem+json rather than letting an unknown
// /api path fall through to the SPA's index.html.
const apiPrefix = "/api/"

// Router builds the application mux with the standard middleware chain already
// applied. Handlers are registered on the returned *http.ServeMux using Go
// 1.22 method+path patterns; API routes live under /api/. The catch-all
// "/api/" is registered here so a misspelled endpoint returns a JSON 404, not
// HTML, before any concrete API handler exists.
type Router struct {
	mux *http.ServeMux
	log *slog.Logger
}

// NewRouter returns a Router with the /api/ catch-all installed.
func NewRouter(log *slog.Logger) *Router {
	mux := http.NewServeMux()
	mux.HandleFunc(apiPrefix, func(w http.ResponseWriter, r *http.Request) {
		WriteProblem(w, CodeNotFound, "Endpoint tidak ditemukan.")
	})
	return &Router{mux: mux, log: log}
}

// Handle registers h for the Go 1.22 method+path pattern (e.g. "GET /api/me").
func (rt *Router) Handle(pattern string, h http.Handler) {
	rt.mux.Handle(pattern, h)
}

// HandleFunc registers a handler func for pattern.
func (rt *Router) HandleFunc(pattern string, h http.HandlerFunc) {
	rt.mux.HandleFunc(pattern, h)
}

// Mux returns the underlying mux for mounting (e.g. behind the static handler).
func (rt *Router) Mux() *http.ServeMux { return rt.mux }

// Handler wraps the mux in the standard middleware chain, outermost first:
// RequestID -> Recover -> Logger -> RealIP.
func (rt *Router) Handler() http.Handler {
	chain := Chain(
		RequestID,
		Recover(rt.log),
		Logger(rt.log),
		RealIP,
	)
	return chain(rt.mux)
}
