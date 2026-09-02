package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/config"
)

// testAccountDomain is the reserved domain every seeded account uses. It is not
// a routable address, so a demo login can never collide with a real person's
// email, and reset:test-data can find every seeded row by joining on it (T075).
const testAccountDomain = "devotion.test"

// demoPassword is the shared login for the synthetic .test fixtures. These
// accounts exist only in a demo or development database (the commands refuse to
// run in production), so a fixed demo secret is documented for the manual test
// script rather than a real credential.
const demoPassword = "sandi-demo-devotion"

// genericBusinessCount is the filler pool that exists only to give search
// results more than one page, so the keyset pagination scenario has something
// to page through.
const genericBusinessCount = 50

// errProductionRefused is returned when either test-data subcommand runs with
// APP_ENV=production. The fixtures carry synthetic identity submissions and
// deadline-bound orders; seeding them beside real data, or truncating real data
// on reset, is exactly what the guard prevents (T075).
var errProductionRefused = errors.New("perintah data uji menolak berjalan saat APP_ENV=production")

// runSeedTestData populates deterministic demo fixtures: a filler pool of
// businesses for search pagination, showcase listings for the matching and
// readiness scenarios, a stale calendar, an expired request, orders already
// sitting in the shipped and late states, pending verifications, and a business
// with too few orders to earn a completion rate. It refuses to run in
// production. Deadline-bound rows are seeded as data already in the target state
// (shipped_at pinned to Clock.Now() minus an offset), never by shifting a clock,
// so nothing here can bleed into a real environment (T075).
func runSeedTestData(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("seed:test-data", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	pool, err := openTestDataPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	res, err := seedTestData(ctx, pool, platform.SystemClock{})
	if err != nil {
		return err
	}
	fmt.Printf("seed:test-data selesai: %d usaha, %d listing, %d pesanan, %d pengajuan verifikasi\n",
		res.Businesses, res.Listings, res.Orders, res.Verifications)
	return nil
}

// runResetTestData empties every table the seeder writes so seed:test-data can
// run again from a clean slate. It refuses to run in production for the same
// reason seed does: a stray invocation there would truncate real data (T075).
func runResetTestData(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("reset:test-data", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	pool, err := openTestDataPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := resetTestData(ctx, pool); err != nil {
		return err
	}
	fmt.Println("reset:test-data selesai")
	return nil
}

// openTestDataPool loads config, refuses when the environment is production, and
// only then opens the pool. The guard runs before any connection is made so a
// misdirected invocation fails fast without touching the database. The caller
// closes the returned pool.
func openTestDataPool(ctx context.Context) (*pgxpool.Pool, error) {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return nil, err
	}
	if err := guardNotProduction(cfg); err != nil {
		return nil, err
	}
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// guardNotProduction returns errProductionRefused when the config is production.
// It is the single decision behind both test-data subcommands, split out so a
// test can assert the refusal without a live database or environment mutation.
func guardNotProduction(cfg config.Config) error {
	if cfg.IsProduction() {
		return errProductionRefused
	}
	return nil
}

// seedResult reports how many fixtures were created, for the command summary and
// for the tests that assert the fixtures landed.
type seedResult struct {
	Businesses    int
	Listings      int
	Orders        int
	Verifications int
}
