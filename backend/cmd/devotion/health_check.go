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
// image. It hits GET /health on the local listener and treats any status other
// than 200 (including 503 from a failing dependency) as unhealthy. The address
// defaults to the plain-HTTP dev port; --addr overrides it.
func runHealthCheck(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("health:check", flag.ExitOnError)
	addr := fs.String("addr", "http://127.0.0.1:8080", "base URL of the running service")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *addr+"/health", nil)
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
