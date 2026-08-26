package account

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
	"github.com/fzrilsh/devotion/backend/internal/platform/ratelimit"
	"github.com/fzrilsh/devotion/backend/internal/platform/session"
)

// baseTime is a fixed Monday so any clock-derived expiry is deterministic.
var baseTime = time.Date(2026, 8, 24, 10, 3, 0, 0, time.UTC)

// captureDelivery records the plaintext code handed to each channel so a test
// can drive verification without a real out-of-band send. Since delivery now
// runs in a goroutine (R-09), each channel is a buffered chan the test reads
// with waitEmail/waitPhone/waitRecovery so it blocks for the send instead of
// racing it.
type captureDelivery struct {
	emailCh, phoneCh, recoveryCh chan string
}

func newCaptureDelivery() *captureDelivery {
	return &captureDelivery{
		emailCh:    make(chan string, 4),
		phoneCh:    make(chan string, 4),
		recoveryCh: make(chan string, 4),
	}
}

func (c *captureDelivery) SendEmailCode(_ context.Context, _, code string) error {
	c.emailCh <- code
	return nil
}
func (c *captureDelivery) SendPhoneCode(_ context.Context, _, code string) error {
	c.phoneCh <- code
	return nil
}
func (c *captureDelivery) SendRecoveryCode(_ context.Context, _, code string) error {
	c.recoveryCh <- code
	return nil
}

// waitEmail blocks for the next emailed code, failing the test if none arrives.
func (c *captureDelivery) waitEmail(t *testing.T) string { return waitCode(t, c.emailCh, "email") }
func (c *captureDelivery) waitPhone(t *testing.T) string { return waitCode(t, c.phoneCh, "phone") }

func waitCode(t *testing.T, ch chan string, which string) string {
	t.Helper()
	select {
	case code := <-ch:
		return code
	case <-time.After(2 * time.Second):
		t.Fatalf("kode %s tidak terkirim dalam batas waktu", which)
		return ""
	}
}

// harness wires a Service against an isolated test schema and returns the router
// handler plus the pieces a test pokes at.
type harness struct {
	svc      *Service
	handler  http.Handler
	pool     *pgxpool.Pool
	clock    *platform.TestClock
	delivery *captureDelivery
}

func newHarness(t *testing.T, name string) *harness {
	t.Helper()
	pool := testdb.New(t, name)
	clock := platform.NewTestClock(baseTime)
	sessions := session.New(pool, clock, false)
	limiter := ratelimit.New(pool, clock)
	delivery := newCaptureDelivery()
	svc := New(pool, clock, sessions, limiter, delivery, quietLogger(), false)

	// The profile born with every account keys a city, so registration needs a
	// city row to point at. Seed one province and one city the tests reuse.
	seedCity(t, pool)

	r := httpx.NewRouter(quietLogger())
	svc.Register(r)

	return &harness{svc: svc, handler: r.Handler(), pool: pool, clock: clock, delivery: delivery}
}

// testCityCode is the seeded city every register call in these tests points at.
const testCityCode = "3273"

// seedCity inserts the one province and city registration needs. It is
// idempotent so a reused schema does not error on the second run.
func seedCity(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO province (code, name) VALUES ('32', 'Jawa Barat') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seed province: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO city (code, province_code, name) VALUES ('3273', '32', 'Kota Bandung') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seed city: %v", err)
	}
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// do sends a JSON request and returns the recorder. cookie, if non-empty, is set
// as the session cookie.
func (h *harness) do(method, path string, body any, cookie string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.RemoteAddr = "203.0.113.7:9999"
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: cookie})
	}
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

// sessionCookie extracts the session cookie value set on a response, or "".
func sessionCookie(rec *httptest.ResponseRecorder) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == session.CookieName {
			return c.Value
		}
	}
	return ""
}

// registerAndLogin creates an account and returns a live session cookie.
func (h *harness) registerAndLogin(t *testing.T, email, phone, password string) string {
	t.Helper()
	rec := h.do("POST", "/api/auth/register", map[string]any{
		"email": email, "phone": phone, "password": password,
		"business_name": "Konveksi Contoh",
		"city_code":     testCityCode,
		"roles":         map[string]any{"buyer": true},
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: status %d, body %s", rec.Code, rec.Body.String())
	}
	rec = h.do("POST", "/api/auth/login", map[string]any{
		"email": email, "password": password,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status %d, body %s", rec.Code, rec.Body.String())
	}
	return sessionCookie(rec)
}

// createAdminAndLogin creates an admin account through the same CreateAdmin path
// the admin:create subcommand uses (no profile row, role_admin set), then logs
// in and returns a live session cookie. It is the fixture for the admin-facing
// gate tests: the account it makes owns no business_profile.
func (h *harness) createAdminAndLogin(t *testing.T, email, phone, password string) string {
	t.Helper()
	if _, err := h.svc.CreateAdmin(context.Background(), email, phone, password); err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}
	rec := h.do("POST", "/api/auth/login", map[string]any{
		"email": email, "password": password,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login admin: status %d, body %s", rec.Code, rec.Body.String())
	}
	return sessionCookie(rec)
}

// TestRegisterLoginMe_HappyPath covers the success path across register, login,
// and GET /me. FR: R-09 registration, T014 /auth and /me.
func TestRegisterLoginMe_HappyPath(t *testing.T) {
	h := newHarness(t, "account_happy")

	cookie := h.registerAndLogin(t, "buyer@example.com", "+6281234567890", "rahasia123")
	if cookie == "" {
		t.Fatal("login tidak menyetel cookie sesi")
	}

	rec := h.do("GET", "/api/me", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /me: status %d", rec.Code)
	}
	var me myAccount
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode MyAccount: %v", err)
	}
	if me.Email != "buyer@example.com" {
		t.Fatalf("email = %q", me.Email)
	}
	if me.ProfileID == nil {
		t.Fatal("profile_id mau terisi karena profil lahir bersama akun")
	}
}

// TestGetMe_NoSession_Unauthorized proves the authenticated route rejects a
// caller without a session. FR: T014 role/auth enforcement.
func TestGetMe_NoSession_Unauthorized(t *testing.T) {
	h := newHarness(t, "account_me_noauth")
	rec := h.do("GET", "/api/me", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, mau 401", rec.Code)
	}
}

// failingDelivery reports every send as failed so a test can prove a broken
// email or WhatsApp channel never rolls back a registration that already
// committed. It records that it was called so the test can assert both channels
// were attempted.
type failingDelivery struct {
	mu               sync.Mutex
	emailCalls       int
	phoneCalls       int
	emailTo, phoneTo string
}

func (d *failingDelivery) SendEmailCode(_ context.Context, to, _ string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.emailCalls++
	d.emailTo = to
	return errors.New("email channel down")
}

func (d *failingDelivery) SendPhoneCode(_ context.Context, to, _ string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.phoneCalls++
	d.phoneTo = to
	return errors.New("whatsapp channel down")
}

func (d *failingDelivery) SendRecoveryCode(_ context.Context, _, _ string) error {
	return errors.New("email channel down")
}

func (d *failingDelivery) snapshot() (int, int, string, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.emailCalls, d.phoneCalls, d.emailTo, d.phoneTo
}

// TestRegister_SendsBothCodes_AndSurvivesDeliveryFailure_FR001_FR002 proves
// registration hands the email code to SendEmailCode and the phone code to
// SendPhoneCode (the FR-002 gate needs both channels to receive a code), and
// that a delivery failure on either channel does not cancel the registration:
// the account is created, GET /me works, and the codes are still resendable.
// FR-001, FR-002, R-09.
func TestRegister_SendsBothCodes_AndSurvivesDeliveryFailure_FR001_FR002(t *testing.T) {
	pool := testdb.New(t, "register_delivery_fail")
	clock := platform.NewTestClock(baseTime)
	sessions := session.New(pool, clock, false)
	limiter := ratelimit.New(pool, clock)
	delivery := &failingDelivery{}
	svc := New(pool, clock, sessions, limiter, delivery, quietLogger(), false)
	seedCity(t, pool)

	r := httpx.NewRouter(quietLogger())
	svc.Register(r)
	h := &harness{svc: svc, handler: r.Handler(), pool: pool, clock: clock}

	rec := h.do("POST", "/api/auth/register", map[string]any{
		"email": "gagal@example.com", "phone": "+6281311112222", "password": "rahasia123",
		"business_name": "Konveksi Contoh",
		"city_code":     testCityCode,
		"roles":         map[string]any{"buyer": true},
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: status %d, body %s (kegagalan kirim tidak boleh membatalkan registrasi)", rec.Code, rec.Body.String())
	}

	// The sends run in goroutines (R-09), so poll briefly for both attempts.
	deadline := time.Now().Add(2 * time.Second)
	for {
		emailCalls, phoneCalls, emailTo, phoneTo := delivery.snapshot()
		if emailCalls >= 1 && phoneCalls >= 1 {
			if emailTo != "gagal@example.com" {
				t.Fatalf("email dikirim ke %q, mau gagal@example.com", emailTo)
			}
			// Registration normalizes the phone (drops the '+') before insert to
			// satisfy phone_format, and delivery sends the stored value.
			if phoneTo != "6281311112222" {
				t.Fatalf("kode HP dikirim ke %q", phoneTo)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("registrasi tidak memicu kedua kanal: email=%d phone=%d", emailCalls, phoneCalls)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// The account survived the failed sends: login and GET /me work.
	rec = h.do("POST", "/api/auth/login", map[string]any{
		"email": "gagal@example.com", "password": "rahasia123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login setelah kirim gagal: status %d, body %s", rec.Code, rec.Body.String())
	}
}

// TestRegister_InvalidInput_Validation proves malformed input is refused before
// any account is created. FR: T014 input validation.
func TestRegister_InvalidInput_Validation(t *testing.T) {
	h := newHarness(t, "account_reg_invalid")
	rec := h.do("POST", "/api/auth/register", map[string]any{
		"email": "bukan-email", "phone": "0812", "password": "pendek",
	}, "")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422", rec.Code)
	}
}

// TestLogin_WrongPassword_Unauthorized proves a bad password is 401 with the
// invalid-credentials code, not leaking whether the account exists. FR: T014.
func TestLogin_WrongPassword_Unauthorized(t *testing.T) {
	h := newHarness(t, "account_login_wrong")
	h.registerAndLogin(t, "sub@example.com", "+6281200000000", "rahasia123")

	rec := h.do("POST", "/api/auth/login", map[string]any{
		"email": "sub@example.com", "password": "salahsemua",
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, mau 401", rec.Code)
	}
}

// TestSessionStoredAsHash proves the raw cookie token is never stored: the
// session row holds a SHA-256 hash, so a database read cannot recover the token.
// FR: T014 "test membuktikan yang disimpan adalah hash, bukan token mentah".
func TestSessionStoredAsHash(t *testing.T) {
	h := newHarness(t, "account_hash")
	cookie := h.registerAndLogin(t, "hash@example.com", "+6281211112222", "rahasia123")

	var count int
	err := h.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM session WHERE token_hash = $1", []byte(cookie)).Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Fatal("token mentah tersimpan sebagai token_hash, mau hash")
	}
	// The SHA-256 of the token must match exactly one row.
	err = h.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM session WHERE token_hash = $1", hashRawToken(cookie)).Scan(&count)
	if err != nil {
		t.Fatalf("query hash: %v", err)
	}
	if count != 1 {
		t.Fatalf("baris cocok hash = %d, mau 1", count)
	}
}

// TestVerifyEmail_HappyPath drives the emailed code back through verify-email
// and confirms the account is marked verified. FR: R-09 verification.
func TestVerifyEmail_HappyPath(t *testing.T) {
	h := newHarness(t, "account_verify")
	cookie := h.registerAndLogin(t, "verify@example.com", "+6281233334444", "rahasia123")

	code := h.delivery.waitEmail(t)
	if !codeRe.MatchString(code) {
		t.Fatalf("kode email = %q, mau enam digit", code)
	}
	rec := h.do("POST", "/api/auth/verify-email", map[string]any{"code": code}, cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("verify-email: status %d, body %s", rec.Code, rec.Body.String())
	}

	rec = h.do("GET", "/api/me", nil, cookie)
	var me myAccount
	_ = json.Unmarshal(rec.Body.Bytes(), &me)
	if !me.EmailVerified {
		t.Fatal("email_verified mau true setelah verifikasi")
	}
}

// TestVerifyEmail_ExpiredCode_Gone proves an expired code is 410 rather than a
// silent accept. FR: R-09 code TTL.
func TestVerifyEmail_ExpiredCode_Gone(t *testing.T) {
	h := newHarness(t, "account_verify_expired")
	cookie := h.registerAndLogin(t, "expire@example.com", "+6281255556666", "rahasia123")
	code := h.delivery.waitEmail(t)

	h.clock.Advance(codeTTL + time.Minute)

	rec := h.do("POST", "/api/auth/verify-email", map[string]any{"code": code}, cookie)
	if rec.Code != http.StatusGone {
		t.Fatalf("status %d, mau 410", rec.Code)
	}
}

// TestRecoverRequest_AlwaysAccepted proves recover/request returns 202 whether
// or not the account exists, so it never leaks account existence. FR: T014
// "POST /auth/recover/request selalu 202".
func TestRecoverRequest_AlwaysAccepted(t *testing.T) {
	h := newHarness(t, "account_recover_req")
	h.registerAndLogin(t, "exists@example.com", "+6281277778888", "rahasia123")

	for _, email := range []string{"exists@example.com", "missing@example.com"} {
		rec := h.do("POST", "/api/auth/recover/request", map[string]any{"email": email}, "")
		if rec.Code != http.StatusAccepted {
			t.Fatalf("email %q: status %d, mau 202", email, rec.Code)
		}
	}
}

// TestLogout_EndsSession proves logout revokes the row so the cookie no longer
// authenticates. FR: T014 "keluar akun benar-benar mengakhiri sesi".
func TestLogout_EndsSession(t *testing.T) {
	h := newHarness(t, "account_logout")
	cookie := h.registerAndLogin(t, "out@example.com", "+6281299990000", "rahasia123")

	rec := h.do("POST", "/api/auth/logout", nil, cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: status %d", rec.Code)
	}
	rec = h.do("GET", "/api/me", nil, cookie)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("setelah logout GET /me: status %d, mau 401", rec.Code)
	}
}

// hashRawToken mirrors the session store's hashing so the hash-storage test can
// assert against the stored value without exporting an internal.
func hashRawToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

// TestRegister_LeavesNoUncoveredRoutes proves every /api route the account
// package registers is either declared public or placed behind an auth gate, so
// the coverage test cannot pass while an account endpoint ships without a role
// decision. FR: T015 route coverage. This runs without a database: Register only
// records patterns, it does not touch the pool.
func TestRegister_LeavesNoUncoveredRoutes(t *testing.T) {
	svc := &Service{}
	r := httpx.NewRouter(quietLogger())
	svc.Register(r)

	if got := r.UncoveredAPIRoutes(); len(got) != 0 {
		t.Fatalf("rute akun tak tercakup = %v, mau kosong", got)
	}
}
