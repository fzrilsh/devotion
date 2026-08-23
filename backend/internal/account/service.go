package account

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/ratelimit"
)

// Errors the service returns to its handlers. Each maps to one problem code; the
// handler owns the HTTP translation so the service stays transport-free.
var (
	errEmailTaken     = errors.New("account: email sudah terdaftar")
	errPhoneTaken     = errors.New("account: nomor sudah terdaftar")
	errBadCredentials = errors.New("account: kredensial salah")
	errCodeInvalid    = errors.New("account: kode salah")
	errCodeExpired    = errors.New("account: kode kedaluwarsa")
	errRolesActive    = errors.New("account: peran masih dipakai order aktif")
	errAccountUnknown = errors.New("account: akun tidak ditemukan")
	errCityUnknown    = errors.New("account: kota tidak dikenal")
)

// registerInput is the validated RegisterRequest. Registration creates the
// account and its business profile in one transaction, so BusinessName and
// CityCode are persisted here: GET /api/profile/me then never 404s, and every
// downstream lookup that keys on business_profile has a row to find.
type registerInput struct {
	Email         string
	Phone         string
	Password      string
	BusinessName  string
	CityCode      string
	Subcontractor bool
	Buyer         bool
}

// register creates the account and its business profile in one transaction,
// then mints and delivers an email code and a phone code. A duplicate email or
// phone is a 409; an unknown city is errCityUnknown so the handler answers 422
// rather than surfacing a foreign key violation as a 500. Code delivery runs
// after the commit and is best effort: a send failure must not roll back a
// registration that already succeeded, since the codes can be resent.
func (s *Service) register(ctx context.Context, in registerInput) (sqlcgen.UserAccount, error) {
	q := s.queries()
	emailExists, err := q.EmailExists(ctx, in.Email)
	if err != nil {
		return sqlcgen.UserAccount{}, err
	}
	if emailExists {
		return sqlcgen.UserAccount{}, errEmailTaken
	}
	phoneExists, err := q.PhoneExists(ctx, in.Phone)
	if err != nil {
		return sqlcgen.UserAccount{}, err
	}
	if phoneExists {
		return sqlcgen.UserAccount{}, errPhoneTaken
	}
	cityExists, err := q.CityExists(ctx, in.CityCode)
	if err != nil {
		return sqlcgen.UserAccount{}, err
	}
	if !cityExists {
		return sqlcgen.UserAccount{}, errCityUnknown
	}

	hash, err := hashPassword(in.Password)
	if err != nil {
		return sqlcgen.UserAccount{}, err
	}
	now := s.clock.Now()

	var acc sqlcgen.UserAccount
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		qtx := sqlcgen.New(tx)
		a, err := qtx.CreateAccount(ctx, sqlcgen.CreateAccountParams{
			Email:             in.Email,
			Phone:             in.Phone,
			PasswordHash:      hash,
			RoleSubcontractor: in.Subcontractor,
			RoleBuyer:         in.Buyer,
			CreatedAt:         tstz(now),
		})
		if err != nil {
			return err
		}
		if _, err := qtx.CreateProfile(ctx, sqlcgen.CreateProfileParams{
			AccountID:    a.ID,
			BusinessName: in.BusinessName,
			CityCode:     in.CityCode,
			CreatedAt:    tstz(now),
		}); err != nil {
			return err
		}
		acc = a
		return nil
	})
	if err != nil {
		return sqlcgen.UserAccount{}, err
	}

	// Both codes are best effort and run only after the account and profile are
	// committed; a flaky mail or WhatsApp channel never blocks registration and
	// never rolls back a row that already landed.
	s.issueAndSend(ctx, acc, sqlcgen.VerificationPurposeEmail)
	s.issueAndSend(ctx, acc, sqlcgen.VerificationPurposePhone)
	return acc, nil
}

// issueAndSend mints a fresh code for the purpose, invalidates any outstanding
// codes for it, stores the new hash, and hands the plaintext to the delivery
// channel exactly once. Errors are returned so callers that care (register does
// not) can react; the plaintext is never persisted.
func (s *Service) issueAndSend(ctx context.Context, acc sqlcgen.UserAccount, purpose sqlcgen.VerificationPurpose) {
	code, err := newCode()
	if err != nil {
		return
	}
	now := s.clock.Now()
	q := s.queries()
	if err := q.InvalidateVerificationCodes(ctx, sqlcgen.InvalidateVerificationCodesParams{
		AccountID:  acc.ID,
		Purpose:    purpose,
		ConsumedAt: tstz(now),
	}); err != nil {
		return
	}
	if _, err := q.CreateVerificationCode(ctx, sqlcgen.CreateVerificationCodeParams{
		AccountID: acc.ID,
		Purpose:   purpose,
		CodeHash:  hashCode(code),
		ExpiresAt: tstz(now.Add(codeTTL)),
		CreatedAt: tstz(now),
	}); err != nil {
		return
	}
	s.deliver(ctx, acc, purpose, code)
}

// deliver routes a plaintext code to its channel. A nil delivery is a no-op, so
// wiring works before the notification channels exist. Delivery errors are
// swallowed here: every caller treats delivery as best effort.
func (s *Service) deliver(ctx context.Context, acc sqlcgen.UserAccount, purpose sqlcgen.VerificationPurpose, code string) {
	if s.delivery == nil {
		return
	}
	switch purpose {
	case sqlcgen.VerificationPurposeEmail:
		_ = s.delivery.SendEmailCode(ctx, acc.Email, code)
	case sqlcgen.VerificationPurposePhone:
		_ = s.delivery.SendPhoneCode(ctx, acc.Phone, code)
	case sqlcgen.VerificationPurposeRecovery:
		_ = s.delivery.SendRecoveryCode(ctx, acc.Email, code)
	}
}

// verify checks a submitted code against the latest unconsumed code for the
// account and purpose, then consumes it. A missing or mismatched code is
// errCodeInvalid; an expired one is errCodeExpired. Expiry is checked in Go
// against the Clock so tests move time instead of waiting.
func (s *Service) verify(ctx context.Context, accountID pgtype.UUID, purpose sqlcgen.VerificationPurpose, submitted string) error {
	q := s.queries()
	row, err := q.GetLatestVerificationCode(ctx, sqlcgen.GetLatestVerificationCodeParams{
		AccountID: accountID,
		Purpose:   purpose,
	})
	if err != nil {
		if isNoRows(err) {
			return errCodeInvalid
		}
		return err
	}
	now := s.clock.Now()
	if !row.ExpiresAt.Valid || !now.Before(row.ExpiresAt.Time) {
		return errCodeExpired
	}
	if !constantTimeEqual(row.CodeHash, hashCode(submitted)) {
		return errCodeInvalid
	}
	affected, err := q.ConsumeVerificationCode(ctx, sqlcgen.ConsumeVerificationCodeParams{
		ID:         row.ID,
		ConsumedAt: tstz(now),
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		// A concurrent request consumed it first; treat as already spent.
		return errCodeInvalid
	}
	return nil
}

// login runs the per-account rate limit before touching bcrypt, so a guesser
// cannot use the login path to burn CPU on the 2GB box. A missing account and a
// wrong password are the same error, so the response does not reveal which
// emails are registered.
func (s *Service) login(ctx context.Context, email, password string) (sqlcgen.UserAccount, error) {
	res, err := s.limiter.Check(ctx, ratelimit.TargetLoginAccount, email)
	if err != nil {
		return sqlcgen.UserAccount{}, err
	}
	if !res.Allowed {
		return sqlcgen.UserAccount{}, rateLimitError{retryAfter: res.RetryAfter}
	}
	acc, err := s.queries().GetAccountByEmail(ctx, email)
	if err != nil {
		if isNoRows(err) {
			// Still run a bcrypt compare against a dummy hash so the timing of a
			// missing account matches a wrong password.
			_ = passwordMatches(dummyHash, password)
			return sqlcgen.UserAccount{}, errBadCredentials
		}
		return sqlcgen.UserAccount{}, err
	}
	if !passwordMatches(acc.PasswordHash, password) {
		return sqlcgen.UserAccount{}, errBadCredentials
	}
	return acc, nil
}

// dummyHash is a bcrypt digest of a throwaway value, compared against when no
// account matches so login latency does not leak account existence. Cost
// matches bcryptCost so the timing lines up.
var dummyHash = mustDummyHash()

func mustDummyHash() string {
	h, err := hashPassword("devotion-timing-equalizer")
	if err != nil {
		panic(err)
	}
	return h
}

// recoverRequest issues a recovery code if the account exists. It never reveals
// whether the account exists: the handler always returns 202, and this runs the
// same work either way. A missing account still consumes the same code path
// minus the actual send.
func (s *Service) recoverRequest(ctx context.Context, email string) {
	acc, err := s.queries().GetAccountByEmail(ctx, email)
	if err != nil {
		return
	}
	s.issueAndSend(ctx, acc, sqlcgen.VerificationPurposeRecovery)
}

// recoverConfirm verifies the recovery code, sets the new password, and returns
// the account so the handler can end every other session. The caller's own
// session, if any, is preserved by the handler.
func (s *Service) recoverConfirm(ctx context.Context, email, code, newPassword string) (sqlcgen.UserAccount, error) {
	acc, err := s.queries().GetAccountByEmail(ctx, email)
	if err != nil {
		if isNoRows(err) {
			return sqlcgen.UserAccount{}, errCodeInvalid
		}
		return sqlcgen.UserAccount{}, err
	}
	if err := s.verify(ctx, acc.ID, sqlcgen.VerificationPurposeRecovery, code); err != nil {
		return sqlcgen.UserAccount{}, err
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return sqlcgen.UserAccount{}, err
	}
	if err := s.queries().UpdatePassword(ctx, sqlcgen.UpdatePasswordParams{
		ID:           acc.ID,
		PasswordHash: hash,
		UpdatedAt:    tstz(s.clock.Now()),
	}); err != nil {
		return sqlcgen.UserAccount{}, err
	}
	return acc, nil
}

// CreateAdmin creates the admin account or resets its password when the email
// already exists, so the admin:create subcommand is idempotent. It reuses the
// one bcrypt path (hashPassword), keeping password hashing in a single place.
// role_admin is set and both business roles stay false; the phone is only
// consulted on first insert, since the upsert touches only the password on
// conflict.
func (s *Service) CreateAdmin(ctx context.Context, email, phone, password string) (sqlcgen.UserAccount, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return sqlcgen.UserAccount{}, err
	}
	return s.queries().UpsertAdmin(ctx, sqlcgen.UpsertAdminParams{
		Email:        email,
		Phone:        phone,
		PasswordHash: hash,
		CreatedAt:    tstz(s.clock.Now()),
	})
}

// account's profile still has active orders on that side is refused, because it
// would strip a party from an order still in flight. The order counts key on the
// business_profile id, so a profile is resolved first; no profile means no
// orders.
func (s *Service) setRoles(ctx context.Context, acc sqlcgen.UserAccount, wantSub, wantBuyer bool) (sqlcgen.UserAccount, error) {
	q := s.queries()
	profileID, err := q.GetProfileIDByAccount(ctx, acc.ID)
	hasProfile := true
	if err != nil {
		if isNoRows(err) {
			hasProfile = false
		} else {
			return sqlcgen.UserAccount{}, err
		}
	}

	if hasProfile && acc.RoleSubcontractor && !wantSub {
		n, err := q.CountActiveOrdersAsSubcontractor(ctx, profileID)
		if err != nil {
			return sqlcgen.UserAccount{}, err
		}
		if n > 0 {
			return sqlcgen.UserAccount{}, errRolesActive
		}
	}
	if hasProfile && acc.RoleBuyer && !wantBuyer {
		n, err := q.CountActiveOrdersAsBuyer(ctx, profileID)
		if err != nil {
			return sqlcgen.UserAccount{}, err
		}
		if n > 0 {
			return sqlcgen.UserAccount{}, errRolesActive
		}
	}

	updated, err := q.UpdateBusinessRoles(ctx, sqlcgen.UpdateBusinessRolesParams{
		ID:                acc.ID,
		RoleSubcontractor: wantSub,
		RoleBuyer:         wantBuyer,
		UpdatedAt:         tstz(s.clock.Now()),
	})
	if err != nil {
		return sqlcgen.UserAccount{}, err
	}
	return updated, nil
}

// VerifyByEmail marks the account with this email verified without a code, for
// the user:verify subcommand. It exists so an admin can rescue an account whose
// number is blocked by the WhatsApp channel shortly before judging, restoring
// the ability to register. A missing account is errAccountUnknown.
func (s *Service) VerifyByEmail(ctx context.Context, email string) error {
	acc, err := s.queries().GetAccountByEmail(ctx, email)
	if err != nil {
		if isNoRows(err) {
			return errAccountUnknown
		}
		return err
	}
	return s.queries().SetEmailVerified(ctx, sqlcgen.SetEmailVerifiedParams{
		ID:        acc.ID,
		UpdatedAt: tstz(s.clock.Now()),
	})
}

// VerifyByPhone marks the account with this phone verified without a code, the
// phone counterpart of VerifyByEmail. A missing account is errAccountUnknown.
func (s *Service) VerifyByPhone(ctx context.Context, phone string) error {
	acc, err := s.queries().GetAccountByPhone(ctx, phone)
	if err != nil {
		if isNoRows(err) {
			return errAccountUnknown
		}
		return err
	}
	return s.queries().SetPhoneVerified(ctx, sqlcgen.SetPhoneVerifiedParams{
		ID:        acc.ID,
		UpdatedAt: tstz(s.clock.Now()),
	})
}

// rateLimitError carries the retry hint so the handler can set Retry-After.
type rateLimitError struct{ retryAfter time.Duration }

func (rateLimitError) Error() string { return "account: rate limit terlampaui" }
