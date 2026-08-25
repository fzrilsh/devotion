package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

var baseTime = time.Date(2026, 8, 23, 9, 0, 0, 0, time.UTC)

// fakePinger reports whichever error it is given, so a test drives both the
// reachable and the down database path without a real pool.
type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

// fakeLink reports a fixed connected bit.
type fakeLink struct{ up bool }

func (l fakeLink) Connected() bool { return l.up }

func quietLogger(t *testing.T) *httpx.Router {
	t.Helper()
	return httpx.NewRouter(httpx.NewLogger())
}

// mkfile writes a file of the given logical size under dir. Truncate sets the
// size without writing content, so a "full" volume costs nothing to simulate.
func mkfile(t *testing.T, dir, name string, size int64) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(size); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func doHealth(t *testing.T, c *Checker) (int, response) {
	t.Helper()
	r := quietLogger(t)
	c.Register(r)
	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)
	var body response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return rec.Code, body
}

// TestHealth_AllUp_200 reports 200 with status ok, a version, and every
// dependency healthy when the database pings, the link is up, and the upload
// volume is far below its quota (an empty temp dir).
func TestHealth_AllUp_200(t *testing.T) {
	c := New(fakePinger{}, fakeLink{up: true}, platform.NewTestClock(baseTime), t.TempDir(), "v-test", 500)
	code, body := doHealth(t, c)
	if code != http.StatusOK {
		t.Fatalf("status %d, mau 200", code)
	}
	if body.Status != "ok" {
		t.Fatalf("status %q, mau ok", body.Status)
	}
	if body.Version != "v-test" {
		t.Fatalf("version %q, mau v-test", body.Version)
	}
	if body.Dependencies.Database != "ok" {
		t.Fatalf("database %q, mau ok", body.Dependencies.Database)
	}
	if body.Dependencies.WhatsApp != "connected" {
		t.Fatalf("whatsapp %q, mau connected", body.Dependencies.WhatsApp)
	}
	if body.Dependencies.Storage.Status != "ok" {
		t.Fatalf("storage %q, mau ok", body.Dependencies.Storage.Status)
	}
	if body.Dependencies.Storage.LimitMB != 500 {
		t.Fatalf("limit_mb %d, mau 500", body.Dependencies.Storage.LimitMB)
	}
	if body.Dependencies.Storage.UsedMB != 0 {
		t.Fatalf("used_mb %d, mau 0", body.Dependencies.Storage.UsedMB)
	}
}

// TestHealth_DBDown_503 reports 503 with status degraded when the database ping
// fails, and the body marks the database fail while the others stay healthy.
func TestHealth_DBDown_503(t *testing.T) {
	c := New(fakePinger{err: errors.New("koneksi ditolak")}, fakeLink{up: true}, platform.NewTestClock(baseTime), t.TempDir(), "v-test", 500)
	code, body := doHealth(t, c)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, mau 503", code)
	}
	if body.Status != "degraded" {
		t.Fatalf("status %q, mau degraded", body.Status)
	}
	if body.Dependencies.Database != "fail" {
		t.Fatalf("database %q, mau fail", body.Dependencies.Database)
	}
	if body.Dependencies.WhatsApp != "connected" {
		t.Fatalf("whatsapp %q, mau connected", body.Dependencies.WhatsApp)
	}
}

// TestHealth_WhatsAppDown_503 reports 503 when the link is down. The body never
// carries the service number (FR-082): the enum has no room for it.
func TestHealth_WhatsAppDown_503_FR082(t *testing.T) {
	c := New(fakePinger{}, fakeLink{up: false}, platform.NewTestClock(baseTime), t.TempDir(), "v-test", 500)
	code, body := doHealth(t, c)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, mau 503", code)
	}
	if body.Status != "degraded" {
		t.Fatalf("status %q, mau degraded", body.Status)
	}
	if body.Dependencies.WhatsApp != "disconnected" {
		t.Fatalf("whatsapp %q, mau disconnected", body.Dependencies.WhatsApp)
	}
}

// TestHealth_StorageFull_503 reports 503 with storage full when used bytes reach
// the total upload quota. An 11MB file against a 10MB limit trips it.
func TestHealth_StorageFull_503(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "besar.bin", 11*1024*1024)
	c := New(fakePinger{}, fakeLink{up: true}, platform.NewTestClock(baseTime), dir, "v-test", 10)
	code, body := doHealth(t, c)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, mau 503", code)
	}
	if body.Status != "degraded" {
		t.Fatalf("status %q, mau degraded", body.Status)
	}
	if body.Dependencies.Storage.Status != "full" {
		t.Fatalf("storage %q, mau full", body.Dependencies.Storage.Status)
	}
}

// TestHealth_StorageNearFull_200 stays healthy but flags near_full once usage
// crosses the warning ratio. 9.6MB against a 10MB limit is 96%.
func TestHealth_StorageNearFull_200(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "hampir.bin", 96*1024*1024/10)
	c := New(fakePinger{}, fakeLink{up: true}, platform.NewTestClock(baseTime), dir, "v-test", 10)
	code, body := doHealth(t, c)
	if code != http.StatusOK {
		t.Fatalf("status %d, mau 200", code)
	}
	if body.Status != "ok" {
		t.Fatalf("status %q, mau ok", body.Status)
	}
	if body.Dependencies.Storage.Status != "near_full" {
		t.Fatalf("storage %q, mau near_full", body.Dependencies.Storage.Status)
	}
}

// TestHealth_BadUploadPath_503 reports storage full and 503 when the upload
// path cannot be read, since new uploads would fail there.
func TestHealth_BadUploadPath_503(t *testing.T) {
	c := New(fakePinger{}, fakeLink{up: true}, platform.NewTestClock(baseTime), "/tidak/ada/path/ini", "v-test", 500)
	code, body := doHealth(t, c)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, mau 503", code)
	}
	if body.Dependencies.Storage.Status != "full" {
		t.Fatalf("storage %q, mau full", body.Dependencies.Storage.Status)
	}
}
