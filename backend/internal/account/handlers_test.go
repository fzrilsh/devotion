package account

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

// captureDelivery records the last plaintext code handed to each channel so a
// test can drive verification without a real out-of-band send.
type captureDelivery struct {
	email, phone, recovery string
}

func (c *captureDelivery) SendEmailCode(_ context.Context, _, code string) error {
	c.email = code
	return nil
}
func (c *captureDelivery) SendPhoneCode(_ context.Context, _, code string) error {
	c.phone = code
	return nil
}
func (c *captureDelivery) SendRecoveryCode(_ context.Context, _, code string) error {
	c.recovery = code
	return nil
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
	delivery := &captureDelivery{}
	svc := New(pool, clock, sessions, limiter, delivery)

	r := httpx.NewRouter(quietLogger())
	svc.Register(r)

	return &harness{svc: svc, handler: r.Handler(), pool: pool, clock: clock, delivery: delivery}
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
		"roles": map[string]any{"buyer": true},
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
	if me.ProfileID != nil {
		t.Fatal("profile_id mau null untuk akun baru")
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

	code := h.delivery.email
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
	code := h.delivery.email

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
