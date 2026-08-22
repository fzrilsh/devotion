package main

import (
	"context"
	"flag"
)

// runAdminCreate creates or updates the admin account, reading the password
// without echo. Implemented in the admin-create branch (T020).
func runAdminCreate(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("admin:create", flag.ExitOnError)
	_ = fs.Parse(args)
	return errNotImplemented
}
