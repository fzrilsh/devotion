package main

import (
	"context"
	"flag"
)

// runSeedTestData populates deterministic test fixtures. Refuses to run when
// APP_ENV is production. Implemented during the user-story phase.
func runSeedTestData(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("seed:test-data", flag.ExitOnError)
	_ = fs.Parse(args)
	return errNotImplemented
}

// runResetTestData truncates test fixtures. Refuses to run when APP_ENV is
// production. Implemented during the user-story phase.
func runResetTestData(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("reset:test-data", flag.ExitOnError)
	_ = fs.Parse(args)
	return errNotImplemented
}
