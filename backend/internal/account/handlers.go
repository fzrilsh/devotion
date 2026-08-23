package account

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
	"github.com/fzrilsh/devotion/backend/internal/platform/ratelimit"
	"github.com/fzrilsh/devotion/backend/internal/platform/session"
)

// Register wires every auth, /me, and /me/roles route onto the router.
//
// The seven routes marked security:[] in the contract are registered with
// Public: they are covered without a role check. register, login, recover, and
// the three verify/resend routes must be reachable before the caller holds a
// usable session (a freshly registered, still-unverified account has a session
// but nothing to lose by verifying).
//
// logout, GET /me, and PATCH /me/roles admit any authenticated caller regardless
// of business role, so they sit behind RequireAuth: the contract documents a 401
// on logout, so it is gated, not public. The gate stores the Principal;
// fromPrincipal pulls the account back out for the /me handlers. logout ignores
// the Principal and revokes by the raw cookie token. Registering them through
// Gated is what keeps them out of the router's uncovered set, so the coverage
// test passes and a future /api route cannot ship without a role decision.
func (s *Service) Register(r *httpx.Router) {
	r.Public("POST /api/auth/register", s.handleRegister)
	r.Public("POST /api/auth/verify-email", s.handleVerifyEmail)
	r.Public("POST /api/auth/verify-phone", s.handleVerifyPhone)
	r.Public("POST /api/auth/resend-code", s.handleResendCode)
	r.Public("POST /api/auth/login", s.handleLogin)
	r.Public("POST /api/auth/recover/request", s.handleRecoverRequest)
	r.Public("POST /api/auth/recover/confirm", s.handleRecoverConfirm)

	auth := httpx.RequireAuth(s)
	r.Gated("POST /api/auth/logout", auth, s.handleLogout)
	r.Gated("GET /api/me", auth, s.fromPrincipal(s.handleGetMe))
	r.Gated("PATCH /api/me/roles", auth, s.fromPrincipal(s.handlePatchRoles))

	r.Gated("GET /api/profile/me", auth, s.fromPrincipal(s.handleGetProfileMe))
	r.Gated("PUT /api/profile/me", auth, s.fromPrincipal(s.handlePutProfileMe))
	r.Public("GET /api/profile/{profileId}", s.handleGetPublicProfile)
}

// authedHandler is a handler that has already resolved the caller's account
// from the session cookie. The httpx auth gate resolves it; fromPrincipal in
// authenticator.go adapts it to this shape.
type authedHandler func(w http.ResponseWriter, r *http.Request, acc sqlcgen.UserAccount)

func (s *Service) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email        string `json:"email"`
		Phone        string `json:"phone"`
		Password     string `json:"password"`
		BusinessName string `json:"business_name"`
		CityCode     string `json:"city_code"`
		Roles        *struct {
			Subcontractor bool `json:"subcontractor"`
			Buyer         bool `json:"buyer"`
		} `json:"roles"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	var fields []httpx.FieldError
	email := normalizeEmail(body.Email)
	if !emailRe.MatchString(email) {
		fields = append(fields, httpx.FieldError{Field: "email", Message: "Format email tidak sah."})
	}
	if !phoneRe.MatchString(body.Phone) {
		fields = append(fields, httpx.FieldError{Field: "phone", Message: "Nomor harus format +62 dan hanya angka."})
	}
	if len(body.Password) < 8 {
		fields = append(fields, httpx.FieldError{Field: "password", Message: "Kata sandi minimal 8 karakter."})
	}
	// business_name is persisted on the profile born with the account, so it is
	// required here. minLength 3 after trim mirrors the business_name_not_empty
	// constraint, turning a would-be 500 into a 422.
	businessName := strings.TrimSpace(body.BusinessName)
	if len(businessName) < 3 {
		fields = append(fields, httpx.FieldError{Field: "business_name", Message: "Nama usaha minimal 3 karakter."})
	}
	// city_code keys the profile's city FK. A four-digit code is required; an
	// unknown one is answered 422 by the service, not a foreign key 500.
	if !cityCodeRe.MatchString(body.CityCode) {
		fields = append(fields, httpx.FieldError{Field: "city_code", Message: "Kode kota harus empat digit angka."})
	}
	// FR-001: registration selects at least one business role. An account with no
	// role would fail the has_at_least_one_role constraint, so reject it here as
	// input rather than surfacing a 500.
	if body.Roles == nil || (!body.Roles.Subcontractor && !body.Roles.Buyer) {
		fields = append(fields, httpx.FieldError{Field: "roles", Message: "Pilih minimal satu peran: subkontraktor atau pemberi order."})
	}
	if len(fields) > 0 {
		httpx.WriteValidation(w, "Masukan tidak sah.", fields)
		return
	}
	in := registerInput{
		Email:        email,
		Phone:        normalizePhone(body.Phone),
		Password:     body.Password,
		BusinessName: businessName,
		CityCode:     body.CityCode,
	}
	if body.Roles != nil {
		in.Subcontractor = body.Roles.Subcontractor
		in.Buyer = body.Roles.Buyer
	}
	_, err := s.register(r.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, errEmailTaken), errors.Is(err, errPhoneTaken):
			httpx.WriteProblem(w, httpx.CodeEmailAlreadyRegistered, "Data sudah terdaftar.")
		case errors.Is(err, errCityUnknown):
			httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
				{Field: "city_code", Message: "Kota tidak dikenal."},
			})
		default:
			httpx.WriteInternal(w)
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"email_verification_required": true,
		"phone_verification_required": true,
	})
}

func (s *Service) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	s.handleVerify(w, r, sqlcgen.VerificationPurposeEmail)
}

func (s *Service) handleVerifyPhone(w http.ResponseWriter, r *http.Request) {
	s.handleVerify(w, r, sqlcgen.VerificationPurposePhone)
}

// handleVerify backs verify-email and verify-phone. Both take an authenticated
// caller (the account being verified is the one holding the session), a code,
// and mark the matching channel verified on success.
func (s *Service) handleVerify(w http.ResponseWriter, r *http.Request, purpose sqlcgen.VerificationPurpose) {
	raw, ok := session.TokenFromRequest(r)
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotAuthenticated, "Belum masuk.")
		return
	}
	sess, err := s.sessions.Validate(r.Context(), raw)
	if err != nil {
		httpx.WriteProblem(w, httpx.CodeNotAuthenticated, "Sesi tidak berlaku. Silakan masuk lagi.")
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.verify(r.Context(), sess.AccountID, purpose, body.Code); err != nil {
		writeCodeError(w, err)
		return
	}
	now := tstz(s.clock.Now())
	if purpose == sqlcgen.VerificationPurposeEmail {
		err = s.queries().SetEmailVerified(r.Context(), sqlcgen.SetEmailVerifiedParams{ID: sess.AccountID, UpdatedAt: now})
	} else {
		err = s.queries().SetPhoneVerified(r.Context(), sqlcgen.SetPhoneVerifiedParams{ID: sess.AccountID, UpdatedAt: now})
	}
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) handleResendCode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Target  string `json:"target"`
		Channel string `json:"channel"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Target == "" || (body.Channel != "email" && body.Channel != "whatsapp") {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Target dan channel wajib diisi.")
		return
	}

	// The per-address distinct-number budget guards a single origin from
	// churning through many numbers; the per-number budget guards one number
	// from repeated presses. Both must pass before a code is issued.
	if addr := clientAddr(r); addr != "" {
		res, err := s.limiter.CheckAddress(r.Context(), addr, body.Target)
		if err != nil {
			httpx.WriteInternal(w)
			return
		}
		if !res.Allowed {
			writeRateLimited(w, res.RetryAfter)
			return
		}
	}
	res, err := s.limiter.Check(r.Context(), ratelimit.TargetOTPPhone, body.Target)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	if !res.Allowed {
		writeRateLimited(w, res.RetryAfter)
		return
	}

	// Look the account up by the target, but never reveal whether it existed:
	// the response is 202 either way.
	var (
		acc     sqlcgen.UserAccount
		found   bool
		purpose sqlcgen.VerificationPurpose
	)
	if body.Channel == "email" {
		a, err := s.queries().GetAccountByEmail(r.Context(), normalizeEmail(body.Target))
		if err == nil {
			acc, found, purpose = a, true, sqlcgen.VerificationPurposeEmail
		} else if !isNoRows(err) {
			httpx.WriteInternal(w)
			return
		}
	} else {
		a, err := s.queries().GetAccountByPhone(r.Context(), body.Target)
		if err == nil {
			acc, found, purpose = a, true, sqlcgen.VerificationPurposePhone
		} else if !isNoRows(err) {
			httpx.WriteInternal(w)
			return
		}
	}
	if found {
		s.issueAndSend(r.Context(), acc, purpose)
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	email := normalizeEmail(body.Email)
	acc, err := s.login(r.Context(), email, body.Password)
	if err != nil {
		var rl rateLimitError
		switch {
		case errors.As(err, &rl):
			writeRateLimited(w, rl.retryAfter)
		case errors.Is(err, errBadCredentials):
			httpx.WriteProblem(w, httpx.CodeInvalidCredentials, "Email atau kata sandi salah.")
		default:
			httpx.WriteInternal(w)
		}
		return
	}
	if err := s.startSession(w, r, acc.ID); err != nil {
		httpx.WriteInternal(w)
		return
	}
	body2, err := s.buildMyAccount(r.Context(), acc)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, body2)
}

func (s *Service) handleLogout(w http.ResponseWriter, r *http.Request) {
	raw, ok := session.TokenFromRequest(r)
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotAuthenticated, "Belum masuk.")
		return
	}
	if err := s.sessions.Revoke(r.Context(), raw); err != nil {
		httpx.WriteInternal(w)
		return
	}
	s.sessions.ClearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) handleRecoverRequest(w http.ResponseWriter, r *http.Request) {
	// Constant response time: the reply is 202 whether or not the account
	// exists, and its duration must not leak what the status code hides. Run the
	// work, then pad to a fixed floor before replying. The floor measures real
	// wall-clock time, not the injected Clock, since the leak it hides is a
	// real-time signal.
	done := platform.ConstantTimeFloor(recoverFloor)
	var body struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	s.recoverRequest(r.Context(), normalizeEmail(body.Email))
	done()
	w.WriteHeader(http.StatusAccepted)
}

func (s *Service) handleRecoverConfirm(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"new_password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	var fields []httpx.FieldError
	if !codeRe.MatchString(body.Code) {
		fields = append(fields, httpx.FieldError{Field: "code", Message: "Kode harus enam digit angka."})
	}
	if len(body.NewPassword) < 8 {
		fields = append(fields, httpx.FieldError{Field: "new_password", Message: "Kata sandi minimal 8 karakter."})
	}
	if len(fields) > 0 {
		httpx.WriteValidation(w, "Masukan tidak sah.", fields)
		return
	}
	acc, err := s.recoverConfirm(r.Context(), normalizeEmail(body.Email), body.Code, body.NewPassword)
	if err != nil {
		writeCodeError(w, err)
		return
	}
	// End every other session after a password reset. The caller's own session,
	// if the request carried one, is preserved so a legitimate reset from a
	// logged-in device is not logged out mid-flow.
	if raw, ok := session.TokenFromRequest(r); ok {
		_ = s.sessions.RevokeOthers(r.Context(), acc.ID, raw)
	} else {
		_ = s.sessions.RevokeAll(r.Context(), acc.ID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) handleGetMe(w http.ResponseWriter, r *http.Request, acc sqlcgen.UserAccount) {
	body, err := s.buildMyAccount(r.Context(), acc)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (s *Service) handlePatchRoles(w http.ResponseWriter, r *http.Request, acc sqlcgen.UserAccount) {
	var body struct {
		Subcontractor bool `json:"subcontractor"`
		Buyer         bool `json:"buyer"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	updated, err := s.setRoles(r.Context(), acc, body.Subcontractor, body.Buyer)
	if err != nil {
		switch {
		case errors.Is(err, errRolesActive):
			httpx.WriteProblem(w, httpx.CodeValidationFailed, "Tidak dapat mencabut peran selama masih ada order aktif.")
		default:
			httpx.WriteInternal(w)
		}
		return
	}
	out, err := s.buildMyAccount(r.Context(), updated)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// startSession issues a session for accountID and sets the cookie. The source
// address (already the true origin after the RealIP middleware) is recorded for
// audit; an unparsable address stores NULL.
func (s *Service) startSession(w http.ResponseWriter, r *http.Request, accountID pgtype.UUID) error {
	raw, err := s.sessions.Issue(r.Context(), accountID, sourceAddr(r))
	if err != nil {
		return err
	}
	s.sessions.SetCookie(w, raw)
	return nil
}

// writeCodeError maps a verification error to its problem code. An expired code
// is 410; anything else in the code path is 422.
func writeCodeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errCodeExpired):
		httpx.WriteProblem(w, httpx.CodeVerificationCodeExpired, "Kode verifikasi kedaluwarsa. Minta kode baru.")
	case errors.Is(err, errCodeInvalid):
		httpx.WriteProblem(w, httpx.CodeInvalidVerificationCode, "Kode verifikasi salah.")
	default:
		httpx.WriteInternal(w)
	}
}

// writeRateLimited writes a 429 with a Retry-After header derived from the
// limiter's hint.
func writeRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	if retryAfter > 0 {
		secs := int(retryAfter.Seconds())
		if secs < 1 {
			secs = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(secs))
	}
	httpx.WriteProblem(w, httpx.CodeRateLimitExceeded, "Terlalu banyak percobaan. Coba lagi nanti.")
}
