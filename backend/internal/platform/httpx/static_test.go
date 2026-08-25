package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// staticFor builds a Static over an in-memory webdist holding index.html plus
// the given extra files, wired to a router so the routing order is exercised
// end to end. register runs against that router before the handler is built.
func staticFor(t *testing.T, extra map[string]string, register func(*Router)) *Static {
	t.Helper()
	files := fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html><title>SPA</title>")},
	}
	for name, body := range extra {
		files[name] = &fstest.MapFile{Data: []byte(body)}
	}
	rt := NewRouter(quietLogger())
	if register != nil {
		register(rt)
	}
	s, err := NewStatic(files, rt.Mux(), rt.Handler())
	if err != nil {
		t.Fatalf("NewStatic: %v", err)
	}
	return s
}

// TestStatic_MuxRouteOutsideAPIReached proves a route registered outside /api/
// is actually served by the mux, not swallowed by the SPA fallback. This is the
// regression guard for the health bug: GET /api/health returned 200 HTML because
// the static handler only routed /api/* to the mux.
func TestStatic_MuxRouteOutsideAPIReached(t *testing.T) {
	s := staticFor(t, nil, func(rt *Router) {
		rt.Public("GET /docs", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "handler")
		})
	})
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest("GET", "/docs", nil))
	if rec.Code != 200 {
		t.Fatalf("status %d, mau 200", rec.Code)
	}
	if body := rec.Body.String(); body != "handler" {
		t.Fatalf("body %q, mau handler (SPA fallback menelan rute)", body)
	}
}

// TestStatic_UnregisteredPathFallsToSPA proves a path with no registered route
// and no matching file still renders index.html, so deep SPA routes survive a
// refresh. This is the counterpart guard: consulting the mux for every path must
// not break the fallback.
func TestStatic_UnregisteredPathFallsToSPA(t *testing.T) {
	s := staticFor(t, nil, nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest("GET", "/pesanan/123", nil))
	if rec.Code != 200 {
		t.Fatalf("status %d, mau 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type %q, mau text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), "SPA") {
		t.Fatalf("body bukan shell SPA: %q", rec.Body.String())
	}
}

// TestStatic_UnknownAPIStaysJSON404 proves an unknown /api path still reaches the
// mux catch-all and gets a problem+json 404, never the SPA shell.
func TestStatic_UnknownAPIStaysJSON404(t *testing.T) {
	s := staticFor(t, nil, nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest("GET", "/api/tidak-ada", nil))
	if rec.Code != 404 {
		t.Fatalf("status %d, mau 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, problemContentType) {
		t.Fatalf("Content-Type %q, mau %q", ct, problemContentType)
	}
}

// TestStatic_RealFileServed proves a hashed asset is served from webdist with the
// immutable cache header, not shadowed by the mux or the fallback.
func TestStatic_RealFileServed(t *testing.T) {
	s := staticFor(t, map[string]string{"assets/app-abc123.js": "console.log(1)"}, nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest("GET", "/assets/app-abc123.js", nil))
	if rec.Code != 200 {
		t.Fatalf("status %d, mau 200", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("Cache-Control %q, mau immutable", cc)
	}
}
