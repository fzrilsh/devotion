package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// healthServer starts a test server that answers GET /api/health with the given
// status code and body, and returns its base URL. Any other path 404s, standing
// in for a listener where only the health route is registered.
func healthServer(t *testing.T, code int, body string) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

// TestHealthCheck_DegradedBodyStays200_Succeeds is the assertion that closes the
// restart loop: health:check must judge the HTTP status code alone, so a 200
// with body status "degraded" (WhatsApp disconnected) is a success. If someone
// later reintroduces a body read demanding "ok", this test goes red before the
// container starts restart-looping a healthy site (R-08).
func TestHealthCheck_DegradedBodyStays200_Succeeds(t *testing.T) {
	addr := healthServer(t, http.StatusOK,
		`{"status":"degraded","dependencies":{"database":"ok","whatsapp":"disconnected","storage":{"status":"ok"}}}`)
	if err := runHealthCheck(context.Background(), []string{"--addr", addr}); err != nil {
		t.Fatalf("health:check gagal pada 200 degraded: %v", err)
	}
}

// TestHealthCheck_OK_Succeeds confirms the ordinary healthy path passes.
func TestHealthCheck_OK_Succeeds(t *testing.T) {
	addr := healthServer(t, http.StatusOK,
		`{"status":"ok","dependencies":{"database":"ok","whatsapp":"connected","storage":{"status":"ok"}}}`)
	if err := runHealthCheck(context.Background(), []string{"--addr", addr}); err != nil {
		t.Fatalf("health:check gagal pada 200 ok: %v", err)
	}
}

// TestHealthCheck_503_Fails confirms a readiness failure (database or storage)
// makes the subcommand exit non-zero so the container healthcheck marks the
// instance unhealthy.
func TestHealthCheck_503_Fails(t *testing.T) {
	addr := healthServer(t, http.StatusServiceUnavailable,
		`{"status":"degraded","dependencies":{"database":"fail","whatsapp":"connected","storage":{"status":"ok"}}}`)
	err := runHealthCheck(context.Background(), []string{"--addr", addr})
	if err == nil {
		t.Fatal("health:check berhasil pada 503, mau gagal")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("galat %q, mau menyebut 503", err.Error())
	}
}
