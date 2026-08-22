package main

import (
	"context"
	"flag"
)

// runHealthCheck probes the running service and exits non-zero on failure, for
// use as a container healthcheck. Implemented in the health-sentry branch
// (T025).
func runHealthCheck(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("health:check", flag.ExitOnError)
	_ = fs.Parse(args)
	return errNotImplemented
}
