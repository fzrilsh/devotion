package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/fzrilsh/devotion/backend/internal/account"
	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/config"
)

// runAdminCreate creates the first admin account, or resets its password when
// the email already exists, so the command is idempotent. The password is read
// from a no-echo prompt rather than a flag, because a flag lands in shell
// history. Email and phone come from flags: they are not secret, and the phone
// is required by the phone_format constraint and phone_unique on the table.
func runAdminCreate(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("admin:create", flag.ExitOnError)
	email := fs.String("email", "", "alamat email admin")
	phone := fs.String("phone", "", "nomor WhatsApp admin, format 62xxxxxxxx")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*email) == "" {
		return errors.New("admin:create butuh --email")
	}
	if strings.TrimSpace(*phone) == "" {
		return errors.New("admin:create butuh --phone")
	}

	password, err := readPassword()
	if err != nil {
		return err
	}

	svc, closePool, err := newAccountService(ctx)
	if err != nil {
		return err
	}
	defer closePool()

	acc, err := svc.CreateAdmin(ctx, strings.TrimSpace(*email), strings.TrimSpace(*phone), password)
	if err != nil {
		return err
	}
	fmt.Printf("admin:create selesai: %s\n", acc.Email)
	return nil
}

// readPassword prompts twice on the terminal without echoing the keystrokes and
// requires the two entries to match, so a mistyped password is caught before it
// is hashed rather than locking the operator out. The plaintext never touches a
// flag, a log, or the returned account.
func readPassword() (string, error) {
	fmt.Print("Kata sandi admin: ")
	first, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	fmt.Print("Ulangi kata sandi: ")
	second, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	if string(first) != string(second) {
		return "", errors.New("kata sandi tidak cocok")
	}
	if len(first) == 0 {
		return "", errors.New("kata sandi kosong")
	}
	return string(first), nil
}

// newAccountService loads config, opens the pool, and builds an account Service
// over a SystemClock. Session store, rate limiter, and delivery are nil: the
// subcommand only creates an admin row and never issues a session, checks a
// budget, or sends a code. The returned func closes the pool.
func newAccountService(ctx context.Context) (*account.Service, func(), error) {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return nil, nil, err
	}
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	svc := account.New(pool, platform.SystemClock{}, nil, nil, nil)
	return svc, pool.Close, nil
}
