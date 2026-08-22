package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

func doHealth(t *testing.T, c *Checker) (int, response) {
	t.Helper()
	r := quietLogger(t)
	c.Register(r)
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)
	var body response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return rec.Code, body
}

// TestHealth_AllUp_200_FR reports 200 with every dependency "ok" when the
// database pings, the link is up, and the upload volume has space. t.TempDir is
// on a real filesystem with far more than one file's worth free.
func TestHealth_AllUp_200(t *testing.T) {
	c := New(fakePinger{}, fakeLink{up: true}, platform.NewTestClock(baseTime), t.TempDir(), 5)
	code, body := doHealth(t, c)
	if code != http.StatusOK {
		t.Fatalf("status %d, mau 200", code)
	}
	if body.Status != "ok" {
		t.Fatalf("status %q, mau ok", body.Status)
	}
	for name, s := range body.Checks {
		if s != stateOK {
			t.Fatalf("check %q = %q, mau ok", name, s)
		}
	}
}

// TestHealth_DBDown_503 reports 503 when the database ping fails, and the body
// names the database as the down dependency while the others stay ok.
func TestHealth_DBDown_503(t *testing.T) {
	c := New(fakePinger{err: errors.New("koneksi ditolak")}, fakeLink{up: true}, platform.NewTestClock(baseTime), t.TempDir(), 5)
	code, body := doHealth(t, c)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, mau 503", code)
	}
	if body.Status != "down" {
		t.Fatalf("status %q, mau down", body.Status)
	}
	if body.Checks["database"] != stateDown {
		t.Fatalf("database = %q, mau down", body.Checks["database"])
	}
	if body.Checks["whatsapp"] != stateOK {
		t.Fatalf("whatsapp = %q, mau ok", body.Checks["whatsapp"])
	}
}

// TestHealth_WhatsAppDown_503 reports 503 when the link is down. The body never
// carries the service number (FR-082): the state enum has no room for it.
func TestHealth_WhatsAppDown_503_FR082(t *testing.T) {
	c := New(fakePinger{}, fakeLink{up: false}, platform.NewTestClock(baseTime), t.TempDir(), 5)
	code, body := doHealth(t, c)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, mau 503", code)
	}
	if body.Checks["whatsapp"] != stateDown {
		t.Fatalf("whatsapp = %q, mau down", body.Checks["whatsapp"])
	}
}

// TestHealth_StorageFull_503 reports 503 when free space is below the floor. A
// floor far larger than any test disk forces the down path deterministically.
func TestHealth_StorageFull_503(t *testing.T) {
	const hugeFloorMB = 1 << 40 // 1 PiB, larger than any CI disk
	c := New(fakePinger{}, fakeLink{up: true}, platform.NewTestClock(baseTime), t.TempDir(), hugeFloorMB)
	code, body := doHealth(t, c)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, mau 503", code)
	}
	if body.Checks["storage"] != stateDown {
		t.Fatalf("storage = %q, mau down", body.Checks["storage"])
	}
}

// TestHealth_BadUploadPath_503 reports storage down when the upload path does
// not exist, since Statfs fails there.
func TestHealth_BadUploadPath_503(t *testing.T) {
	c := New(fakePinger{}, fakeLink{up: true}, platform.NewTestClock(baseTime), "/tidak/ada/path/ini", 5)
	code, body := doHealth(t, c)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, mau 503", code)
	}
	if body.Checks["storage"] != stateDown {
		t.Fatalf("storage = %q, mau down", body.Checks["storage"])
	}
}
