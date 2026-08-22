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
//  1. /api/* is handled by the mux (the API handler and the /api/ catch-all that
//     returns a JSON 404 for unknown API paths, never index.html).
//  2. A real file under webdist is served with a long Cache-Control, because
//     Vite emits content-hashed asset names that are safe to cache forever.
//  3. Every other path falls back to index.html with Cache-Control: no-cache, so
//     a refresh on a deep SPA route does not 404.
//
// The order is the point: an unknown /api path must reach the mux's catch-all
// and get a problem+json 404, not the SPA shell, or a mistyped endpoint returns
// HTML and misleads diagnosis.
type Static struct {
	api      http.Handler
	files    http.Handler
	fsys     fs.FS
	indexRaw []byte
}

// NewStatic builds the static handler over dist, a filesystem already rooted at
// the directory that holds index.html (use fs.Sub on the embed.FS with the
// webdist prefix before passing it here). api handles everything under /api/.
func NewStatic(dist fs.FS, api http.Handler) (*Static, error) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, err
	}
	return &Static{
		api:      api,
		files:    http.FileServer(http.FS(dist)),
		fsys:     dist,
		indexRaw: index,
	}, nil
}

func (s *Static) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, apiPrefix) {
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
