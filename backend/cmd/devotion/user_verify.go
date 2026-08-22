package main

import (
	"context"
	"flag"
)

// runUserVerify verifies an account without a UI, by email or phone, so a
// number blocked shortly before judging does not remove the ability to create
// accounts. Implemented in the whatsapp-admin branch (T024a).
func runUserVerify(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("user:verify", flag.ExitOnError)
	_ = fs.String("email", "", "email akun yang diverifikasi")
	_ = fs.String("phone", "", "nomor telepon akun yang diverifikasi")
	_ = fs.Parse(args)
	return errNotImplemented
}
