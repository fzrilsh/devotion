package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// baseTime is a fixed Monday used as the clock origin so window boundaries are
// deterministic across runs.
var baseTime = time.Date(2026, 8, 24, 10, 3, 0, 0, time.UTC)

// TestCheck_LoginAccount_FifthPassesSixthBlocked proves the login limit is 5
// per 15 minutes per account and that the sixth attempt is refused with a
// Retry-After that points at the window rollover. FR: R-10 login window.
func TestCheck_LoginAccount_FifthPassesSixthBlocked(t *testing.T) {
	pool := testdb.New(t, "ratelimit_login")
	clock := platform.NewTestClock(baseTime)
	l := New(pool, clock)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		res, err := l.Check(ctx, TargetLoginAccount, "akun-1")
		if err != nil {
			t.Fatalf("percobaan %d: %v", i, err)
		}
		if !res.Allowed {
			t.Fatalf("percobaan %d ditolak, mau lolos", i)
		}
	}
	res, err := l.Check(ctx, TargetLoginAccount, "akun-1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Allowed {
		t.Fatal("percobaan ke-6 lolos, mau ditolak")
	}
	if res.RetryAfter <= 0 || res.RetryAfter > 15*time.Minute {
		t.Fatalf("Retry-After = %v, mau (0, 15m]", res.RetryAfter)
	}
}

// TestCheck_WindowExpiryResets moves the clock past the window instead of
// sleeping and proves the counter resets. FR: R-10 fixed window.
func TestCheck_WindowExpiryResets(t *testing.T) {
	pool := testdb.New(t, "ratelimit_expiry")
	clock := platform.NewTestClock(baseTime)
	l := New(pool, clock)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if _, err := l.Check(ctx, TargetLoginAccount, "akun-2"); err != nil {
			t.Fatal(err)
		}
	}
	if res, _ := l.Check(ctx, TargetLoginAccount, "akun-2"); res.Allowed {
		t.Fatal("ke-6 lolos sebelum jendela habis")
	}

	clock.Advance(16 * time.Minute)
	res, err := l.Check(ctx, TargetLoginAccount, "akun-2")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Allowed {
		t.Fatal("jendela baru menolak, mau lolos")
	}
}

// TestCheck_KeysAreIndependent proves two accounts do not share a budget.
func TestCheck_KeysAreIndependent(t *testing.T) {
	pool := testdb.New(t, "ratelimit_keys")
	clock := platform.NewTestClock(baseTime)
	l := New(pool, clock)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if _, err := l.Check(ctx, TargetLoginAccount, "akun-a"); err != nil {
			t.Fatal(err)
		}
	}
	res, err := l.Check(ctx, TargetLoginAccount, "akun-b")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Allowed {
		t.Fatal("akun-b ditolak karena kuota akun-a, mau independen")
	}
}

// TestCheck_OTPPhone_ThirdPassesFourthBlocked proves the per-number code limit
// is 3 per hour. FR: R-10 otp_phone window.
func TestCheck_OTPPhone_ThirdPassesFourthBlocked(t *testing.T) {
	pool := testdb.New(t, "ratelimit_otpphone")
	clock := platform.NewTestClock(baseTime)
	l := New(pool, clock)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		res, err := l.Check(ctx, TargetOTPPhone, "628123")
		if err != nil {
			t.Fatalf("kirim %d: %v", i, err)
		}
		if !res.Allowed {
			t.Fatalf("kirim %d ditolak, mau lolos", i)
		}
	}
	res, _ := l.Check(ctx, TargetOTPPhone, "628123")
	if res.Allowed {
		t.Fatal("kirim ke-4 lolos, mau ditolak")
	}
	if res.RetryAfter <= 0 || res.RetryAfter > time.Hour {
		t.Fatalf("Retry-After = %v, mau (0, 1h]", res.RetryAfter)
	}
}

// TestCheck_QuotaRequest_TwentiethPassesTwentyFirstBlocked proves the per-user
// quota-request limit is 20 per hour. FR: R-10 quota_request window.
func TestCheck_QuotaRequest_TwentiethPassesTwentyFirstBlocked(t *testing.T) {
	pool := testdb.New(t, "ratelimit_quota")
	clock := platform.NewTestClock(baseTime)
	l := New(pool, clock)
	ctx := context.Background()

	for i := 1; i <= 20; i++ {
		res, err := l.Check(ctx, TargetQuotaRequest, "user-1")
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		if !res.Allowed {
			t.Fatalf("request %d ditolak, mau lolos", i)
		}
	}
	if res, _ := l.Check(ctx, TargetQuotaRequest, "user-1"); res.Allowed {
		t.Fatal("request ke-21 lolos, mau ditolak")
	}
}

// TestCheckAddress_CountsDistinctNumbers proves the per-address limit counts
// distinct numbers, not attempts: ten different numbers pass, the eleventh is
// blocked, but a re-send to an already-counted number still passes because it
// is not a new distinct number. FR: R-10 otp_address window.
func TestCheckAddress_CountsDistinctNumbers(t *testing.T) {
	pool := testdb.New(t, "ratelimit_otpaddr")
	clock := platform.NewTestClock(baseTime)
	l := New(pool, clock)
	ctx := context.Background()
	const addr = "203.0.113.5"

	for i := 0; i < 10; i++ {
		number := "62800000000" + string(rune('0'+i))
		res, err := l.CheckAddress(ctx, addr, number)
		if err != nil {
			t.Fatalf("nomor %d: %v", i, err)
		}
		if !res.Allowed {
			t.Fatalf("nomor berbeda ke-%d ditolak, mau lolos", i+1)
		}
	}

	// Re-sending to a number already counted is not a new distinct number.
	res, err := l.CheckAddress(ctx, addr, "628000000000")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Allowed {
		t.Fatal("kirim ulang ke nomor lama ditolak, mau lolos (bukan nomor baru)")
	}

	// An eleventh distinct number exceeds the address budget.
	res, err = l.CheckAddress(ctx, addr, "62899999999")
	if err != nil {
		t.Fatal(err)
	}
	if res.Allowed {
		t.Fatal("nomor berbeda ke-11 lolos, mau ditolak")
	}
	if res.RetryAfter <= 0 || res.RetryAfter > time.Hour {
		t.Fatalf("Retry-After = %v, mau (0, 1h]", res.RetryAfter)
	}
}

// TestCheckAddress_DifferentAddressesIndependent proves one address filling its
// distinct-number budget does not block another address.
func TestCheckAddress_DifferentAddressesIndependent(t *testing.T) {
	pool := testdb.New(t, "ratelimit_otpaddr_indep")
	clock := platform.NewTestClock(baseTime)
	l := New(pool, clock)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		number := "62811111111" + string(rune('0'+i))
		if _, err := l.CheckAddress(ctx, "198.51.100.1", number); err != nil {
			t.Fatal(err)
		}
	}
	res, err := l.CheckAddress(ctx, "198.51.100.2", "62822222222")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Allowed {
		t.Fatal("alamat kedua ditolak karena kuota alamat pertama, mau independen")
	}
}
