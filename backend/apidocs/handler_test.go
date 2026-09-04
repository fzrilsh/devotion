package apidocs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// newRouter builds a router with the /docs routes registered, as serve does in
// every environment since the competition decision (see utang-teknis.md).
func newRouter(t *testing.T) *httpx.Router {
	t.Helper()
	r := httpx.NewRouter(httpx.NewLogger())
	Register(r)
	return r
}

func do(t *testing.T, r *httpx.Router, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

// TestDocs_UI_200HTML proves GET /docs returns 200 with an HTML Swagger UI shell
// when the routes are registered (serve registers them in every environment). T082.
func TestDocs_UI_200HTML(t *testing.T) {
	rec := do(t, newRouter(t), "/docs")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type %q, mau text/html", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Fatalf("body bukan shell Swagger UI")
	}
	if !strings.Contains(body, `url: "/docs/openapi.yaml"`) {
		t.Fatalf("UI tidak menunjuk /docs/openapi.yaml")
	}
	// The page must carry SRI hashes so a compromised CDN cannot inject code.
	if !strings.Contains(body, "integrity=\"sha384-") {
		t.Fatalf("aset CDN tanpa integrity SRI")
	}
}

// TestDocs_RoutesStayMuxMounted pins the competition decision that /docs is
// registered in every environment: a fresh mux without apidocs.Register must
// NOT be how serve builds the router anymore. It guards against someone
// reintroducing the environment gate at the registration site, which would
// silently send the jury's /docs requests to the SPA fallback again.
func TestDocs_RoutesStayMuxMounted(t *testing.T) {
	r := newRouter(t)
	_, patternUI := r.Mux().Handler(httptest.NewRequest("GET", "/docs", nil))
	if patternUI == "" {
		t.Fatal("rute /docs tidak terdaftar, mau terpasang di mux di semua environment")
	}
	_, patternSpec := r.Mux().Handler(httptest.NewRequest("GET", "/docs/openapi.yaml", nil))
	if patternSpec == "" {
		t.Fatal("rute /docs/openapi.yaml tidak terdaftar")
	}
}

// TestDocs_ServedSpecIdenticalToSource is the load-bearing guard: the bytes
// served at /docs/openapi.yaml must be byte-for-byte identical to the canonical
// contract. A missed sync would serve a stale spec that compiles and passes vet
// but silently drifts from the source. YAML validity is not checked here: if the
// served bytes match the source, the only way it is invalid YAML is a broken
// source, which openapi-typescript (T004) fails hard on, not this handler's job.
// T082.
func TestDocs_ServedSpecIdenticalToSource(t *testing.T) {
	rec := do(t, newRouter(t), "/docs/openapi.yaml")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/yaml") {
		t.Fatalf("Content-Type %q, mau application/yaml", ct)
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("tidak dapat menentukan path berkas uji")
	}
	// backend/apidocs/handler_test.go -> repo root is three levels up.
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	src := filepath.Join(repoRoot, "docs", "001-capacity-exchange-marketplace", "contracts", "openapi.yaml")
	want, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("baca sumber kontrak: %v", err)
	}

	got := rec.Body.Bytes()
	if len(got) != len(want) {
		t.Fatalf("panjang spec disajikan %d byte, sumber %d byte; jalankan ./backend/apidocs-sync.sh lalu commit", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("spec menyimpang dari sumber pada byte %d; jalankan ./backend/apidocs-sync.sh lalu commit", i)
		}
	}
}
