package account

import (
	"errors"
	"net/http"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
	"github.com/fzrilsh/devotion/backend/internal/platform/session"
)

// Authenticate resolves the session cookie to the caller's Principal, satisfying
// httpx.Authenticator so the httpx auth gates can enforce roles without importing
// this package. It validates the token, loads the account fresh (so a role change
// or a verification takes effect on the next request), and folds the three role
// flags into the bitmask httpx checks. The opaque UserAccount rides along in
// Principal.Account for handlers that need the full row.
//
// An absent, invalid, or dangling session returns httpx.ErrUnauthenticated, which
// the gate turns into a 401; a database error is returned as-is so the gate emits
// a 500 rather than mistaking a hiccup for a logged-out caller.
func (s *Service) Authenticate(r *http.Request) (httpx.Principal, error) {
	raw, ok := session.TokenFromRequest(r)
	if !ok {
		return httpx.Principal{}, httpx.ErrUnauthenticated
	}
	sess, err := s.sessions.Validate(r.Context(), raw)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return httpx.Principal{}, httpx.ErrUnauthenticated
		}
		return httpx.Principal{}, err
	}
	acc, err := s.queries().GetAccountByID(r.Context(), sess.AccountID)
	if err != nil {
		if isNoRows(err) {
			return httpx.Principal{}, httpx.ErrUnauthenticated
		}
		return httpx.Principal{}, err
	}
	return httpx.Principal{Roles: roleMask(acc), Account: acc}, nil
}

// roleMask folds the three boolean role flags on the account row into the httpx
// bitmask. The admin bit is separate from the two business roles, never implied
// by them.
func roleMask(acc sqlcgen.UserAccount) httpx.Role {
	var m httpx.Role
	if acc.RoleSubcontractor {
		m |= httpx.RoleSubcontractor
	}
	if acc.RoleBuyer {
		m |= httpx.RoleBuyer
	}
	if acc.RoleAdmin {
		m |= httpx.RoleAdmin
	}
	return m
}

// fromPrincipal adapts an authedHandler to run behind an httpx auth gate: the
// gate has already stored the Principal, so this pulls the account back out and
// hands it to h. A missing Principal is an invariant violation (the route was
// gated), so it is a 500 rather than a silent 401.
func (s *Service) fromPrincipal(h authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := httpx.PrincipalFromContext(r.Context())
		if !ok {
			httpx.WriteInternal(w)
			return
		}
		acc, ok := p.Account.(sqlcgen.UserAccount)
		if !ok {
			httpx.WriteInternal(w)
			return
		}
		h(w, r, acc)
	}
}
