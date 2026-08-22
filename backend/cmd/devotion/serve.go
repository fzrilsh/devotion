package main

import (
	"context"
	"flag"
)

// runServe boots the HTTP server: load config, connect the pool, run
// migrations, start the scheduler goroutine, then listen. Implemented in the
// embed-serving-tls branch (T022).
func runServe(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	_ = fs.Parse(args)
	return errNotImplemented
}
