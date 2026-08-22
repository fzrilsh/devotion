package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// stubAuth is a test Authenticator: it returns the configured principal and
// error verbatim, so a test can drive every branch of the auth gates without a
// session store or database.
type stubAuth struct {
	principal Principal
	err       error
}

func (a stubAuth) Authenticate(*http.Request) (Principal, error) {
	return a.principal, a.err
}

// TestRequireAuth_RejectsAbsentSession proves an ErrUnauthenticated from the
// Authenticator becomes a 401 CodeNotAuthenticated and the wrapped handler never
// runs.
func TestRequireAuth_RejectsAbsentSession(t *testing.T) {
	ran := false
	h := RequireAuth(stubAuth{err: ErrUnauthenticated})(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) { ran = true }))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/me", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
	if ran {
		t.Fatal("handler jalan meski tidak terautentikasi")
	}
}

// TestRequireAuth_ResolutionErrorIs500 proves a non-ErrUnauthenticated error (a
// database hiccup) becomes a 500, never a 401, so a transient failure is not
// mistaken for a logged-out caller.
func TestRequireAuth_ResolutionErrorIs500(t *testing.T) {
	h := RequireAuth(stubAuth{err: errors.New("db down")})(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/me", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, mau 500", rec.Code)
	}
}

// TestRequireAuth_AdmitsAndStoresPrincipal proves an authenticated caller reaches
// the handler with the Principal on the context.
func TestRequireAuth_AdmitsAndStoresPrincipal(t *testing.T) {
	want := Principal{Roles: RoleBuyer, Account: "acc"}
	var got Principal
	var ok bool
	h := RequireAuth(stubAuth{principal: want})(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			got, ok = PrincipalFromContext(r.Context())
		}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/me", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}
	if !ok || got.Roles != want.Roles || got.Account != want.Account {
		t.Fatalf("principal = %+v ok=%v, mau %+v", got, ok, want)
	}
}

// TestRequireRole_RejectionMatrix drives every unauthorized role combination: a
// caller whose mask shares no bit with the required set gets a 403, and a caller
// holding any required bit is admitted. This is the T015 obligation that a route
// rejects callers of the wrong role.
func TestRequireRole_RejectionMatrix(t *testing.T) {
	cases := []struct {
		name     string
		have     Role
		require  []Role
		wantCode int
	}{
		{"buyer-only ke rute subkontraktor", RoleBuyer, []Role{RoleSubcontractor}, http.StatusForbidden},
		{"subkontraktor ke rute buyer", RoleSubcontractor, []Role{RoleBuyer}, http.StatusForbidden},
		{"buyer ke rute admin", RoleBuyer, []Role{RoleAdmin}, http.StatusForbidden},
		{"tanpa peran ke rute subkontraktor", 0, []Role{RoleSubcontractor}, http.StatusForbidden},
		{"buyer ke rute buyer", RoleBuyer, []Role{RoleBuyer}, http.StatusOK},
		{"admin ke rute admin", RoleAdmin, []Role{RoleAdmin}, http.StatusOK},
		{"dua peran usaha ke rute buyer", RoleBuyer | RoleSubcontractor, []Role{RoleBuyer}, http.StatusOK},
		{"buyer ke rute buyer-atau-subkontraktor", RoleBuyer, []Role{RoleSubcontractor, RoleBuyer}, http.StatusOK},
		{"admin bukan peran usaha", RoleAdmin, []Role{RoleBuyer, RoleSubcontractor}, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ran := false
			h := RequireRole(stubAuth{principal: Principal{Roles: tc.have}}, tc.require...)(
				http.HandlerFunc(func(http.ResponseWriter, *http.Request) { ran = true }))

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/x", nil))

			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, mau %d", rec.Code, tc.wantCode)
			}
			if ran != (tc.wantCode == http.StatusOK) {
				t.Fatalf("handler jalan=%v, tapi status %d", ran, rec.Code)
			}
		})
	}
}

// TestRequireRole_AuthFailureBefore403 proves the auth check precedes the role
// check: an unauthenticated caller gets 401, not 403, so a logged-out user is
// told to log in rather than that they are forbidden.
func TestRequireRole_AuthFailureBefore403(t *testing.T) {
	h := RequireRole(stubAuth{err: ErrUnauthenticated}, RoleAdmin)(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/admin", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
}

// TestRouter_NoUncoveredAPIRoutes proves the coverage bookkeeping: a plain
// Handle on an /api route shows up as uncovered, while Public and Gated do not,
// and the bare /api/ catch-all is exempt. This is the mechanism that fails the
// build when an endpoint ships without a role decision.
func TestRouter_NoUncoveredAPIRoutes(t *testing.T) {
	rt := NewRouter(quietLogger())
	rt.Public("POST /api/auth/login", func(http.ResponseWriter, *http.Request) {})
	gate := RequireRole(stubAuth{}, RoleAdmin)
	rt.Gated("GET /api/admin/whatsapp", gate, func(http.ResponseWriter, *http.Request) {})

	if got := rt.UncoveredAPIRoutes(); len(got) != 0 {
		t.Fatalf("rute tak tercakup = %v, mau kosong", got)
	}

	// A plain Handle on an /api route must be reported, so a new endpoint cannot
	// slip through without a role decision.
	rt.HandleFunc("GET /api/listings", func(http.ResponseWriter, *http.Request) {})
	got := rt.UncoveredAPIRoutes()
	if len(got) != 1 || got[0] != "GET /api/listings" {
		t.Fatalf("tak tercakup = %v, mau [GET /api/listings]", got)
	}
}
