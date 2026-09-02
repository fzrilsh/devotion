package httpx

import (
	"bytes"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

// Static serves the embedded SPA with the routing order fixed by research R-06:
//
//  1. Any path with a registered mux route (every /api/* path via the /api/
//     catch-all, plus routes mounted outside /api/ such as /api/health and, in
//     development, /docs) is handled by the mux through the middleware chain.
//  2. A real file under webdist is served with a long Cache-Control, because
//     Vite emits content-hashed asset names that are safe to cache forever.
//  3. Every other path falls back to index.html with Cache-Control: no-cache, so
//     a refresh on a deep SPA route does not 404.
//
// The order is the point: the mux is consulted for every path before the SPA
// fallback, so a route registered outside /api/ is actually reached instead of
// being swallowed by index.html. An unknown /api path still reaches the mux's
// catch-all and gets a problem+json 404, not the SPA shell, or a mistyped
// endpoint returns HTML and misleads diagnosis.
type Static struct {
	mux      *http.ServeMux
	api      http.Handler
	files    http.Handler
	fsys     fs.FS
	indexRaw []byte
}

// NewStatic builds the static handler over dist, a filesystem already rooted at
// the directory that holds index.html (use fs.Sub on the embed.FS with the
// webdist prefix before passing it here). mux is the raw router mux, consulted
// to decide whether a path has a registered route; api is that same mux wrapped
// in the middleware chain and is what actually serves a matched route.
func NewStatic(dist fs.FS, mux *http.ServeMux, api http.Handler) (*Static, error) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, err
	}
	return &Static{
		mux:      mux,
		api:      api,
		files:    http.FileServer(http.FS(dist)),
		fsys:     dist,
		indexRaw: index,
	}, nil
}

func (s *Static) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Consult the mux for every path, not just /api/*. ServeMux.Handler returns a
	// non-empty pattern only when a route is registered for the request, so a
	// match means a real handler (an API route, the /api/ catch-all, /api/health,
	// or /docs) rather than the SPA shell.
	if _, pattern := s.mux.Handler(r); pattern != "" {
		s.api.ServeHTTP(w, r)
		return
	}

	if p := strings.TrimPrefix(r.URL.Path, "/"); p != "" && fileExists(s.fsys, p) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		s.files.ServeHTTP(w, r)
		return
	}

	// SPA fallback: any non-file, non-API path renders the shell so client-side
	// routing can take over. no-cache keeps a stale shell from pinning an old
	// asset manifest after a deploy.
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(s.indexRaw))
}

// fileExists reports whether name is a regular file in fsys. A directory is not
// a servable file: without this, a request for "/" or a folder name would match
// and the http.FileServer would emit a directory listing instead of the SPA.
func fileExists(fsys fs.FS, name string) bool {
	f, err := fs.Stat(fsys, name)
	if err != nil {
		return false
	}
	return !f.IsDir()
}
