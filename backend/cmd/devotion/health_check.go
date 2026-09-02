package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"
)

// runHealthCheck probes the running service over HTTP and exits non-zero on
// failure, so the container healthcheck can call it without shipping curl in the
// image. It hits GET /api/health on the local listener and judges the HTTP
// status code alone: 200 means the process is alive and serving, whatever the
// body's status field says, and 503 means a readiness dependency (database or
// storage) failed. A "degraded" body still returns 200, so WhatsApp being
// disconnected does not fail the check and cannot drive a restart loop: the
// route is registered, so a 200 always comes from the health handler, never the
// SPA fallback. The address defaults to the plain HTTP dev port; --addr
// overrides it.
func runHealthCheck(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("health:check", flag.ExitOnError)
	addr := fs.String("addr", "http://127.0.0.1:8080", "base URL of the running service")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *addr+"/api/health", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check gagal: status %d", resp.StatusCode)
	}
	return nil
}
