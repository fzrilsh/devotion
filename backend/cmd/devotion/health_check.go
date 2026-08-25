package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"
)

// runHealthCheck probes the running service over HTTP and exits non-zero on
// failure, so the container healthcheck can call it without shipping curl in the
// image. It hits GET /api/health on the local listener and treats any status
// other than 200 (including 503 from a failing dependency) as unhealthy. It also
// decodes the body and requires status "ok": a 200 with an HTML body (the SPA
// shell) or a missing status field means the probe hit the wrong handler, so it
// fails rather than reporting a false healthy. The address defaults to the plain
// HTTP dev port; --addr overrides it.
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

	// A 200 alone is not enough: the SPA fallback also returns 200. Require a JSON
	// body reporting status "ok" so a probe that lands on the wrong handler fails.
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("health check gagal: body bukan JSON: %w", err)
	}
	if body.Status != "ok" {
		return fmt.Errorf("health check gagal: status %q", body.Status)
	}
	return nil
}
