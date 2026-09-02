package admin

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// stubAuth maps any request to a fixed Principal. roles is the bitmask the route
// gate checks; fail forces an unauthenticated result so the 401 path is exercised.
type stubAuth struct {
	roles httpx.Role
	fail  bool
}

func (a stubAuth) Authenticate(*http.Request) (httpx.Principal, error) {
	if a.fail {
		return httpx.Principal{}, httpx.ErrUnauthenticated
	}
	return httpx.Principal{Roles: a.roles}, nil
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// newManager builds a Manager with no whatsmeow client, enough to drive the
// handler: Status is nil-safe on the client, so the guarded fields alone
// determine the body. It stands in for a real manager whose link is down.
func newManager(qr, lastErr string) *Manager {
	m := &Manager{log: quietLogger()}
	m.qrCode = qr
	m.lastError = lastErr
	return m
}

// TestStatus_AdminOnly proves the WhatsApp status route is gated to admins: a
// buyer is 403, an unauthenticated caller is 401, and an admin gets 200. FR-082
// still holds regardless of role, since the number is never a body field.
func TestStatus_AdminOnly_FR082(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		auth   stubAuth
		want   int
	}{
		{"status unauthenticated", "GET", "/api/admin/whatsapp", stubAuth{fail: true}, http.StatusUnauthorized},
		{"status buyer forbidden", "GET", "/api/admin/whatsapp", stubAuth{roles: httpx.RoleBuyer}, http.StatusForbidden},
		{"status admin ok", "GET", "/api/admin/whatsapp", stubAuth{roles: httpx.RoleAdmin}, http.StatusOK},
		{"reconnect unauthenticated", "POST", "/api/admin/whatsapp/reconnect", stubAuth{fail: true}, http.StatusUnauthorized},
		{"reconnect subcontractor forbidden", "POST", "/api/admin/whatsapp/reconnect", stubAuth{roles: httpx.RoleSubcontractor}, http.StatusForbidden},
		{"reconnect admin ok", "POST", "/api/admin/whatsapp/reconnect", stubAuth{roles: httpx.RoleAdmin}, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httpx.NewRouter(quietLogger())
			newManager("", "").Register(r, tc.auth)
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			r.Handler().ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status %d, mau %d", rec.Code, tc.want)
			}
		})
	}
}

// TestStatus_NullsWhenEmpty proves qr_code and last_error render as JSON null
// when unset, matching the nullable contract fields, and that connected is false
// for a manager with no live client. FR-082.
func TestStatus_NullsWhenEmpty_FR082(t *testing.T) {
	r := httpx.NewRouter(quietLogger())
	newManager("", "").Register(r, stubAuth{roles: httpx.RoleAdmin})
	req := httptest.NewRequest("GET", "/api/admin/whatsapp", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)

	var body whatsAppStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Connected {
		t.Fatal("connected mau false tanpa client")
	}
	if body.QRCode != nil || body.LastError != nil {
		t.Fatalf("qr_code/last_error mau null, dapat %+v", body)
	}
}

// TestStatus_CarriesQRAndError proves a pending QR and a recorded error reach the
// body, so the admin page can render a code to scan or the last failure. The
// service number is never among these fields (FR-082).
func TestStatus_CarriesQRAndError_FR082(t *testing.T) {
	r := httpx.NewRouter(quietLogger())
	newManager("qr-opaque", "sesi WhatsApp keluar, pindai ulang kode QR").Register(r, stubAuth{roles: httpx.RoleAdmin})
	req := httptest.NewRequest("GET", "/api/admin/whatsapp", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)

	var body whatsAppStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.QRCode == nil || *body.QRCode != "qr-opaque" {
		t.Fatalf("qr_code = %v, mau qr-opaque", body.QRCode)
	}
	if body.LastError == nil || *body.LastError == "" {
		t.Fatal("last_error mau terisi")
	}
}
