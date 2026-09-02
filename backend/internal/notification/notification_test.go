package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
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
)

// baseTime is a fixed instant so every clock-derived timestamp in these tests is
// deterministic; the notification feed orders on created_at, so a fixed clock
// keeps the keyset order reproducible.
var baseTime = time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)

// stubAuth is the notification tests' Authenticator: it maps any request to a
// fixed Principal carrying the seeded account, standing in for the account
// service the real wiring passes. The routes only need an authenticated caller,
// so a constant principal is enough to exercise them.
type stubAuth struct {
	acc  sqlcgen.UserAccount
	fail bool
}

func (a stubAuth) Authenticate(*http.Request) (httpx.Principal, error) {
	if a.fail {
		return httpx.Principal{}, httpx.ErrUnauthenticated
	}
	return httpx.Principal{Roles: httpx.RoleBuyer, Account: a.acc}, nil
}

// okSender records the messages it was handed and always succeeds.
type okSender struct{ sent int }

func (s *okSender) Send(context.Context, string, string, string) error { s.sent++; return nil }
func (s *okSender) SendText(context.Context, string, string) error     { s.sent++; return nil }

// errSender always fails, standing in for an unavailable transport so a channel
// exhausts its attempts.
type errSender struct{}

func (errSender) Send(context.Context, string, string, string) error { return errors.New("kirim gagal") }
func (errSender) SendText(context.Context, string, string) error     { return errors.New("kirim gagal") }

// harness wires a notification Service against an isolated test schema.
type harness struct {
	svc     *Service
	handler http.Handler
	pool    *pgxpool.Pool
	clock   *platform.TestClock
	acc     pgtype.UUID
}

func newHarness(t *testing.T, name string) *harness {
	t.Helper()
	pool := testdb.New(t, name)
	clock := platform.NewTestClock(baseTime)
	accID := seedAccount(t, pool, "user@example.com", "628123456789")

	svc := New(pool, clock, stubAuth{acc: sqlcgen.UserAccount{ID: accID}}, nil, nil)
	r := httpx.NewRouter(quietLogger())
	svc.Register(r)

	return &harness{svc: svc, handler: r.Handler(), pool: pool, clock: clock, acc: accID}
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// seedAccount inserts one account and returns its id. The buyer role satisfies
// has_at_least_one_role; the phone matches phone_format.
func seedAccount(t *testing.T, pool *pgxpool.Pool, email, phone string) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO user_account (email, phone, password_hash, role_buyer, created_at, updated_at)
		 VALUES ($1, $2, 'x', true, $3, $3) RETURNING id`,
		email, phone, baseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}
	return id
}

// do sends a JSON request through the router and returns the recorder.
func (h *harness) do(method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}
