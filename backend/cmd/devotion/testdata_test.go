package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/order"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/config"
)

// fixedSeedInstant is a Monday 09:00 WIB so WeekStart lands on the same day and
// the deadline-bound fixtures resolve to a known instant regardless of when the
// suite runs. Using an injected clock, not time.Now(), is what keeps the shipped
// and late orders on a fixed side of their windows (Rule 5, T075).
var fixedSeedInstant = time.Date(2026, 3, 2, 9, 0, 0, 0, platform.Jakarta)

// TestGuardNotProduction_RefusesProduction_T075 proves both test-data
// subcommands refuse to run under APP_ENV=production. The guard takes a Config
// directly, so the refusal is asserted without a live database or mutating
// process environment.
func TestGuardNotProduction_RefusesProduction_T075(t *testing.T) {
	prod := config.Config{AppEnv: config.EnvProduction}
	if err := guardNotProduction(prod); !errors.Is(err, errProductionRefused) {
		t.Fatalf("guardNotProduction(production) = %v, ingin errProductionRefused", err)
	}

	for _, env := range []config.Env{config.EnvDevelopment, config.EnvTest} {
		cfg := config.Config{AppEnv: env}
		if err := guardNotProduction(cfg); err != nil {
			t.Errorf("guardNotProduction(%s) = %v, ingin nil", env, err)
		}
	}
}

// TestSeedTestData_ProducesFixtures_T075 runs the seeder against an isolated
// migrated schema with an injected clock, then asserts the required fixtures
// landed: the filler pool plus showcase listings, the deadline-bound orders
// seeded already in their target state, the pending verifications, and the
// business with too few orders to earn a completion rate.
func TestSeedTestData_ProducesFixtures_T075(t *testing.T) {
	pool := testdb.New(t, "seed_test_data")
	ctx := context.Background()
	clock := platform.NewTestClock(fixedSeedInstant)

	res, err := seedTestData(ctx, pool, clock)
	if err != nil {
		t.Fatalf("seedTestData: %v", err)
	}

	// Every showcase and generic listing counts one business, the three shipped
	// or late orders count two parties each, the few-orders business adds its
	// subcontractor plus two buyers, and the expired-request buyer adds one.
	if res.Businesses < genericBusinessCount {
		t.Errorf("Businesses = %d, ingin minimal %d", res.Businesses, genericBusinessCount)
	}
	if res.Verifications != 2 {
		t.Errorf("Verifications = %d, ingin 2", res.Verifications)
	}
	// dikirim-6-hari, dikirim-8-hari, pesanan-telat, plus the two orders of the
	// few-orders business.
	if res.Orders != 5 {
		t.Errorf("Orders = %d, ingin 5", res.Orders)
	}

	// 50 generic + 4 showcase + 1 stale-calendar + 1 expired-request buyer +
	// 3 shipped/late orders at two parties each + 2 verifications + the
	// few-orders subcontractor and its 2 buyers.
	const wantAccounts = genericBusinessCount + 4 + 1 + 1 + 3*2 + 2 + 3
	assertCount(t, ctx, pool, "usaha", wantAccounts,
		`SELECT count(*) FROM user_account`)

	// The shipped-8-days order sits past the 7-day auto-confirm window; the
	// shipped-6-days order sits inside it. Both are already in the shipped state,
	// pinned by shipped_at, never by advancing a clock. The due/not-due split is
	// asserted through the domain function in its own test below.
	assertCount(t, ctx, pool, "pesanan dikirim", 2,
		`SELECT count(*) FROM work_order WHERE status = 'shipped'`)

	// The late order's deadline fell before today's WIB day and it is still
	// active, so it shows on the admin late list.
	assertOne(t, ctx, pool, "pesanan telat",
		`SELECT count(*) FROM work_order
		  WHERE status = 'production' AND deadline < $1`,
		fixedSeedInstant)

	// The expired request lapsed without a reply: its 72-hour window closed and
	// it carries no candidate.
	assertOne(t, ctx, pool, "request kedaluwarsa",
		`SELECT count(*) FROM quota_request qr
		  WHERE qr.reply_due_at < $1
		    AND NOT EXISTS (SELECT 1 FROM request_candidate rc WHERE rc.request_id = qr.id)`,
		fixedSeedInstant)

	// The stale-calendar listing has not been touched for eight days, past the
	// seven-day staleness window.
	assertOne(t, ctx, pool, "kalender basi",
		`SELECT count(*) FROM capacity_listing
		  WHERE calendar_updated_at <= $1`,
		fixedSeedInstant.Add(-8*24*time.Hour))

	// Two pending verification submissions await moderation.
	assertCount(t, ctx, pool, "verifikasi menunggu", 2,
		`SELECT count(*) FROM verification_request WHERE status = 'pending'`)

	// The few-orders subcontractor carries exactly two confirmed orders, one
	// short of the completion-rate threshold.
	assertOne(t, ctx, pool, "usaha dua pesanan",
		`SELECT count(*) FROM (
		    SELECT subcontractor_id
		      FROM work_order
		     WHERE status = 'confirmed'
		     GROUP BY subcontractor_id
		    HAVING count(*) = 2
		 ) t`)
}

// TestSeedTestData_AccountsUseTestDomain_T075 asserts every seeded account uses
// the reserved .test domain, so a demo login can never collide with a real
// person's address (T075 "akun uji memakai domain .test").
func TestSeedTestData_AccountsUseTestDomain_T075(t *testing.T) {
	pool := testdb.New(t, "seed_domain")
	ctx := context.Background()

	if _, err := seedTestData(ctx, pool, platform.NewTestClock(fixedSeedInstant)); err != nil {
		t.Fatalf("seedTestData: %v", err)
	}

	var stray int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM user_account WHERE email NOT LIKE $1`,
		"%@"+testAccountDomain).Scan(&stray); err != nil {
		t.Fatalf("hitung email non-.test: %v", err)
	}
	if stray != 0 {
		t.Errorf("%d akun tidak memakai domain %s", stray, testAccountDomain)
	}
	if !strings.HasSuffix(testAccountDomain, ".test") {
		t.Errorf("testAccountDomain = %q, ingin berakhiran .test", testAccountDomain)
	}
}

// TestResetTestData_ClearsFixtures_T075 seeds, resets, and asserts the fixture
// tables are empty while the region and catalog reference rows survive, so
// seed:test-data can run again from a clean slate.
func TestResetTestData_ClearsFixtures_T075(t *testing.T) {
	pool := testdb.New(t, "reset_test_data")
	ctx := context.Background()

	if _, err := seedTestData(ctx, pool, platform.NewTestClock(fixedSeedInstant)); err != nil {
		t.Fatalf("seedTestData: %v", err)
	}
	if err := resetTestData(ctx, pool); err != nil {
		t.Fatalf("resetTestData: %v", err)
	}

	for _, table := range resetFixtureTables {
		assertCount(t, ctx, pool, table+" setelah reset", 0,
			"SELECT count(*) FROM "+table)
	}

	// Region and catalog reference rows are deliberately preserved so a rerun of
	// the seeder does not have to rebuild them.
	assertNonZero(t, ctx, pool, "province", `SELECT count(*) FROM province`)
	assertNonZero(t, ctx, pool, "catalog_item", `SELECT count(*) FROM catalog_item`)
}

// TestSeedTestData_ShippedOrderAutoConfirmWindow_FR068 proves the two shipped
// fixtures land on the correct sides of the auto-confirm window and, crucially,
// checks that split through the same order.IsAutoConfirmDue the scheduler and
// the read layer use, not an ad-hoc query. The dikirim-6-hari order must not yet
// be due; the dikirim-8-hari order must be due. Both are read back with their
// stored shipped_at and evaluated against the injected clock's instant.
func TestSeedTestData_ShippedOrderAutoConfirmWindow_FR068(t *testing.T) {
	pool := testdb.New(t, "seed_autoconfirm")
	ctx := context.Background()
	now := fixedSeedInstant

	if _, err := seedTestData(ctx, pool, platform.NewTestClock(now)); err != nil {
		t.Fatalf("seedTestData: %v", err)
	}

	cases := []struct {
		emailPrefix string
		wantDue     bool
	}{
		{"dikirim-6-hari-pembeli", false},
		{"dikirim-8-hari-pembeli", true},
	}
	for _, tc := range cases {
		var baseAt, shippedAt pgtype.Timestamptz
		err := pool.QueryRow(ctx,
			`SELECT wo.auto_confirm_base_at, wo.shipped_at
			   FROM work_order wo
			   JOIN business_profile pr ON pr.id = wo.buyer_id
			   JOIN user_account ua ON ua.id = pr.account_id
			  WHERE ua.email = $1`,
			tc.emailPrefix+"@"+testAccountDomain).Scan(&baseAt, &shippedAt)
		if err != nil {
			t.Fatalf("%s: baca work_order: %v", tc.emailPrefix, err)
		}
		base := order.AutoConfirmBase(baseAt, shippedAt)
		got := order.IsAutoConfirmDue(base, now, false)
		if got != tc.wantDue {
			t.Errorf("%s: IsAutoConfirmDue = %v, ingin %v (base %s, now %s)",
				tc.emailPrefix, got, tc.wantDue, base, now)
		}
	}
}

// TestSeedTestData_Idempotent_T075 runs the seeder twice against one schema and
// asserts the second run neither errors nor doubles the fixtures, the same
// re-runnability seed:regions and seed:master-data give.
func TestSeedTestData_Idempotent_T075(t *testing.T) {
	pool := testdb.New(t, "seed_idempotent")
	ctx := context.Background()
	clock := platform.NewTestClock(fixedSeedInstant)

	first, err := seedTestData(ctx, pool, clock)
	if err != nil {
		t.Fatalf("seedTestData (pertama): %v", err)
	}
	second, err := seedTestData(ctx, pool, clock)
	if err != nil {
		t.Fatalf("seedTestData (kedua): %v", err)
	}
	if first != second {
		t.Errorf("hasil seed berbeda antar jalanan: %+v vs %+v", first, second)
	}

	const wantAccounts = genericBusinessCount + 4 + 1 + 1 + 3*2 + 2 + 3
	assertCount(t, ctx, pool, "usaha setelah dua kali seed", wantAccounts,
		`SELECT count(*) FROM user_account`)
}

// TestSeedTestData_CapacityCandidateMeetsLargeOrder_SC019 proves the seeded
// 500/minggu jeda 0 hari candidate really clears the capacity criterion for a
// 3.000-piece order eight weeks out: 500 x (this week through deadline week) is
// at least 3.000, the sum-across-periods SC-019 pins down.
func TestSeedTestData_CapacityCandidateMeetsLargeOrder_SC019(t *testing.T) {
	pool := testdb.New(t, "seed_capacity")
	ctx := context.Background()
	now := fixedSeedInstant
	thisWeek := platform.WeekStart(now)
	deadlineWeek := thisWeek.AddDate(0, 0, 7*8) // eight weeks out

	if _, err := seedTestData(ctx, pool, platform.NewTestClock(now)); err != nil {
		t.Fatalf("seedTestData: %v", err)
	}

	const wantQuantity = 3000
	var remaining int64
	err := pool.QueryRow(ctx,
		`SELECT coalesce(sum(ap.total_capacity - ap.used_capacity), 0)
		   FROM capacity_listing l
		   JOIN business_profile pr ON pr.id = l.profile_id
		   JOIN user_account ua ON ua.id = pr.account_id
		   JOIN availability_period ap ON ap.listing_id = l.id
		  WHERE ua.email = $1
		    AND NOT ap.marked_full
		    AND ap.week_start BETWEEN $2 AND $3`,
		"kapasitas-500@"+testAccountDomain,
		thisWeek, deadlineWeek).Scan(&remaining)
	if err != nil {
		t.Fatalf("hitung kapasitas kandidat 500: %v", err)
	}
	if remaining < wantQuantity {
		t.Errorf("kapasitas terjumlah = %d, ingin minimal %d untuk lolos kriteria",
			remaining, wantQuantity)
	}
}

func assertCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, label string, want int, query string, args ...any) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, query, args...).Scan(&got); err != nil {
		t.Fatalf("%s: query: %v", label, err)
	}
	if got != want {
		t.Errorf("%s = %d, ingin %d", label, got, want)
	}
}

func assertOne(t *testing.T, ctx context.Context, pool *pgxpool.Pool, label string, query string, args ...any) {
	t.Helper()
	assertCount(t, ctx, pool, label, 1, query, args...)
}

func assertNonZero(t *testing.T, ctx context.Context, pool *pgxpool.Pool, label string, query string, args ...any) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, query, args...).Scan(&got); err != nil {
		t.Fatalf("%s: query: %v", label, err)
	}
	if got == 0 {
		t.Errorf("%s = 0, ingin lebih dari nol", label)
	}
}
