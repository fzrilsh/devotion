package httpx

import (
	"context"
	"errors"
	"net/http"
	"slices"
)

// Role is a single capability bit. An account carries a bitmask of these: the
// two business roles may be held together, and the admin bit is separate from
// both (an admin is not a buyer or subcontractor by virtue of being an admin).
// The set matches the boolean flags on the account row; httpx keeps them as a
// bitmask so a route can require any of several roles without the account
// package leaking into this layer.
type Role uint8

const (
	// RoleSubcontractor may publish capacity listings and accept quota requests.
	RoleSubcontractor Role = 1 << iota
	// RoleBuyer may send quota requests against others' listings.
	RoleBuyer
	// RoleAdmin may reach the moderation, verification, and mediation surface.
	// It is a separate bit, never implied by a business role.
	RoleAdmin
)

// Has reports whether r contains want.
func (r Role) Has(want Role) bool { return r&want != 0 }

// Principal is the authenticated caller: the role bitmask a route gate checks,
// plus an opaque Account the owning package stashed for its own handlers.
// httpx never inspects Account; the account package type-asserts it back.
type Principal struct {
	Roles   Role
	Account any
}

// ErrUnauthenticated is what an Authenticator returns when the request carries
// no valid session. The auth middleware turns it into a 401; any other error is
// a 500, so a database hiccup is never mistaken for an absent session.
var ErrUnauthenticated = errors.New("httpx: tidak terautentikasi")

// Authenticator resolves a request to its Principal. The account package
// implements it (validate the session cookie, load the account, read its
// roles), so httpx enforces auth and role gates without importing account and
// creating a cycle.
type Authenticator interface {
	Authenticate(r *http.Request) (Principal, error)
}

// principalKey holds the authenticated Principal on the request context. It is
// set to 1 explicitly rather than via iota so it cannot collide with
// requestIDKey (0) as keys are added across files in this package.
const principalKey ctxKey = 1

// withPrincipal returns a copy of ctx carrying p.
func withPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// PrincipalFromContext returns the Principal placed on ctx by RequireAuth or
// RequireRole. The second result is false when no auth middleware ran, which a
// handler behind one of those gates can treat as an invariant violation.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

// RequireAuth admits any authenticated caller and stores the Principal on the
// context for the wrapped handler. An absent or invalid session is 401; a
// resolution error (database, session store) is 500.
func RequireAuth(auth Authenticator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, err := auth.Authenticate(r)
			if err != nil {
				if errors.Is(err, ErrUnauthenticated) {
					WriteProblem(w, CodeNotAuthenticated, "Sesi tidak berlaku. Silakan masuk lagi.")
					return
				}
				WriteInternal(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), p)))
		})
	}
}

// RequireRole admits an authenticated caller holding at least one of roles.
// With no roles listed it is exactly RequireAuth. Authentication failure is
// 401; a valid caller without a listed role is 403, so a logged-in user hitting
// an endpoint meant for another role is told they are forbidden, not that they
// are logged out.
func RequireRole(auth Authenticator, roles ...Role) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, err := auth.Authenticate(r)
			if err != nil {
				if errors.Is(err, ErrUnauthenticated) {
					WriteProblem(w, CodeNotAuthenticated, "Sesi tidak berlaku. Silakan masuk lagi.")
					return
				}
				WriteInternal(w)
				return
			}
			if !hasAnyRole(p.Roles, roles) {
				WriteProblem(w, CodeForbidden, "Anda tidak berwenang mengakses ini.")
				return
			}
			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), p)))
		})
	}
}

// hasAnyRole reports whether have contains any of want. An empty want passes:
// RequireRole with no roles is just an authentication gate.
func hasAnyRole(have Role, want []Role) bool {
	if len(want) == 0 {
		return true
	}
	return slices.ContainsFunc(want, have.Has)
}
