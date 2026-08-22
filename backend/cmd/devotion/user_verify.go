package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
)

// runUserVerify verifies an account without a UI, by email or phone, so a
// number blocked shortly before judging does not remove the ability to create
// accounts (T024a). Exactly one of --email or --phone is required; passing both
// or neither is an error, since each names a different lookup.
func runUserVerify(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("user:verify", flag.ExitOnError)
	email := fs.String("email", "", "email akun yang diverifikasi")
	phone := fs.String("phone", "", "nomor telepon akun yang diverifikasi")
	if err := fs.Parse(args); err != nil {
		return err
	}

	e, p := strings.TrimSpace(*email), strings.TrimSpace(*phone)
	if (e == "") == (p == "") {
		return errors.New("user:verify butuh tepat satu dari --email atau --phone")
	}

	svc, closePool, err := newAccountService(ctx)
	if err != nil {
		return err
	}
	defer closePool()

	if e != "" {
		if err := svc.VerifyByEmail(ctx, e); err != nil {
			return err
		}
		fmt.Printf("user:verify selesai: email %s terverifikasi\n", e)
		return nil
	}
	if err := svc.VerifyByPhone(ctx, p); err != nil {
		return err
	}
	fmt.Printf("user:verify selesai: nomor %s terverifikasi\n", p)
	return nil
}
