package httpx

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"
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
//
// Router also records, per registered pattern, whether it was declared public
// or placed behind an auth gate. UncoveredAPIRoutes reports any /api/ pattern
// that is neither, so a test can fail the build when an endpoint ships without a
// role decision. The constitution treats an endpoint with no role check as
// defective, and this is the mechanism that enforces it.
type Router struct {
	mux    *http.ServeMux
	log    *slog.Logger
	routes map[string]routeGate
}

// routeGate records how a pattern was registered: explicitly public, or behind
// an auth gate. A pattern registered through plain Handle/HandleFunc has both
// false and shows up as uncovered.
type routeGate struct {
	public bool
	gated  bool
}

// NewRouter returns a Router with the /api/ catch-all installed.
func NewRouter(log *slog.Logger) *Router {
	mux := http.NewServeMux()
	mux.HandleFunc(apiPrefix, func(w http.ResponseWriter, r *http.Request) {
		WriteProblem(w, CodeNotFound, "Endpoint tidak ditemukan.")
	})
	return &Router{mux: mux, log: log, routes: make(map[string]routeGate)}
}

// Handle registers h for the Go 1.22 method+path pattern (e.g. "GET /api/me").
// It records no gate: an /api/ pattern registered this way is reported by
// UncoveredAPIRoutes. Use Public or Gated for API routes.
func (rt *Router) Handle(pattern string, h http.Handler) {
	rt.record(pattern, routeGate{})
	rt.mux.Handle(pattern, h)
}

// HandleFunc registers a handler func for pattern with no gate. See Handle.
func (rt *Router) HandleFunc(pattern string, h http.HandlerFunc) {
	rt.record(pattern, routeGate{})
	rt.mux.HandleFunc(pattern, h)
}

// record stores the gate for pattern, lazily allocating the map so a Router
// built by direct struct literal (as some tests do) does not panic.
func (rt *Router) record(pattern string, g routeGate) {
	if rt.routes == nil {
		rt.routes = make(map[string]routeGate)
	}
	rt.routes[pattern] = g
}

// Public registers h for pattern and marks it explicitly public. Use this for
// the auth routes that carry security:[] in the contract (register, login,
// recover), so they are covered without a role check.
func (rt *Router) Public(pattern string, h http.HandlerFunc) {
	rt.record(pattern, routeGate{public: true})
	rt.mux.HandleFunc(pattern, h)
}

// Gated registers h for pattern wrapped in mw (a RequireAuth or RequireRole
// gate) and marks it covered. The gate runs before h, so h can assume an
// authenticated Principal on the context.
func (rt *Router) Gated(pattern string, mw Middleware, h http.HandlerFunc) {
	rt.record(pattern, routeGate{gated: true})
	rt.mux.Handle(pattern, mw(h))
}

// UncoveredAPIRoutes returns the /api/ patterns that are neither declared public
// nor placed behind an auth gate, sorted for a stable failure message. The bare
// /api/ catch-all is exempt: it only ever returns 404. A test asserts this is
// empty so a new endpoint cannot ship without a role decision.
func (rt *Router) UncoveredAPIRoutes() []string {
	var out []string
	for pattern, g := range rt.routes {
		path := patternPath(pattern)
		if !strings.HasPrefix(path, apiPrefix) || path == apiPrefix {
			continue
		}
		if !g.public && !g.gated {
			out = append(out, pattern)
		}
	}
	sort.Strings(out)
	return out
}

// patternPath returns the path portion of a Go 1.22 route pattern, dropping a
// leading method and optional host. "GET /api/me" yields "/api/me".
func patternPath(pattern string) string {
	if i := strings.IndexByte(pattern, '/'); i >= 0 {
		return pattern[i:]
	}
	return pattern
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
