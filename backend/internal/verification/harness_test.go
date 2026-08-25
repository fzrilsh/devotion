package verification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
	"github.com/fzrilsh/devotion/backend/internal/platform/storage"
)

// baseTime is a fixed instant so any clock-derived created_at is deterministic.
var baseTime = time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)

var phoneSeq int

func nextPhone() string {
	phoneSeq++
	return fmt.Sprintf("62813%07d", phoneSeq)
}

// mockAuth is a stand-in Authenticator so the tests never import account.
type mockAuth struct {
	principal *httpx.Principal
}

func (m *mockAuth) Authenticate(_ *http.Request) (httpx.Principal, error) {
	if m.principal == nil {
		return httpx.Principal{}, httpx.ErrUnauthenticated
	}
	return *m.principal, nil
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type harness struct {
	svc     *Service
	handler http.Handler
	pool    *pgxpool.Pool
	auth    *mockAuth
	acc     pgtype.UUID
	profile pgtype.UUID
}

// newHarness builds a pool, storage service, verification service, and router,
// then authenticates as a buyer whose profile owns nothing yet. Individual
// tests flip auth.principal to exercise the role and ownership branches.
func newHarness(t *testing.T, name string) *harness {
	t.Helper()
	pool := testdb.New(t, name)
	clock := platform.NewTestClock(baseTime)
	store, err := storage.New(pool, clock, t.TempDir(), 5, 500)
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	svc := New(pool, clock, store)

	acc, prof := seedProfile(t, pool, "buyer@contoh.test", false)
	auth := &mockAuth{principal: &httpx.Principal{
		Roles:   httpx.RoleBuyer,
		Account: sqlcgen.UserAccount{ID: acc},
	}}

	r := httpx.NewRouter(quietLogger())
	svc.Register(r, auth)

	return &harness{
		svc: svc, handler: r.Handler(), pool: pool, auth: auth,
		acc: acc, profile: prof,
	}
}

// seedProfile inserts the region, account, and business_profile a file or a
// verification row references, returning the account and profile ids.
func seedProfile(t *testing.T, pool *pgxpool.Pool, email string, subcontractor bool) (pgtype.UUID, pgtype.UUID) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO province (code, name) VALUES ('32', 'Jawa Barat') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seed province: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO city (code, province_code, name) VALUES ('3273', '32', 'Bandung') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seed city: %v", err)
	}
	var accountID pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO user_account (email, phone, password_hash, role_subcontractor, role_buyer, created_at, updated_at)
		 VALUES ($1, $2, 'x', $3, $4, $5, $5) RETURNING id`,
		email, nextPhone(), subcontractor, !subcontractor, baseTime).Scan(&accountID); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	var profileID pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO business_profile (account_id, business_name, city_code, verified, created_at, updated_at)
		 VALUES ($1, 'Konveksi Uji', '3273', false, $2, $2) RETURNING id`,
		accountID, baseTime).Scan(&profileID); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	return accountID, profileID
}

// seedAdmin inserts an admin account (no business_profile), returning its id.
func seedAdmin(t *testing.T, pool *pgxpool.Pool, email string) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO user_account (email, phone, password_hash, role_admin, created_at, updated_at)
		 VALUES ($1, $2, 'x', true, $3, $3) RETURNING id`,
		email, nextPhone(), baseTime).Scan(&id); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	return id
}

// jpegBytes returns a valid one-pixel JPEG so an upload drives the real decode
// and re-encode path in storage.
func jpegBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func (h *harness) asPrincipal(p *httpx.Principal) { h.auth.principal = p }

func (h *harness) doJSON(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.7:9999"
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

func (h *harness) do(method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = "203.0.113.7:9999"
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

// uploadMultipart posts a file part plus a kind field and returns the recorder.
func (h *harness) uploadMultipart(t *testing.T, kind string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("kind", kind); err != nil {
		t.Fatalf("write kind: %v", err)
	}
	part, err := mw.CreateFormFile("file", "berkas.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/files", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.RemoteAddr = "203.0.113.7:9999"
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

func mustStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status %d, mau %d; body %s", rec.Code, want, rec.Body.String())
	}
}

type problemResp struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Errors []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"errors"`
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) problemResp {
	t.Helper()
	var p problemResp
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem %q: %v", rec.Body.String(), err)
	}
	return p
}

func decodeUpload(t *testing.T, rec *httptest.ResponseRecorder) uploadResponse {
	t.Helper()
	var u uploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
		t.Fatalf("decode upload %q: %v", rec.Body.String(), err)
	}
	return u
}
