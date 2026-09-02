package account

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"math/big"

	"github.com/jackc/pgx/v5"
)

// codeDigits is the length of a verification or recovery code. R-09 fixes six
// digits; the contract's VerificationCodeRequest accepts 4 to 8 so a future
// length change needs no contract edit, but everything this service mints is
// six.
const codeDigits = 6

// newCode returns a fresh numeric code of codeDigits digits, drawn from
// crypto/rand so it is not predictable. Leading zeros are kept, so the space is
// the full 10^codeDigits and a code like "000123" is valid.
func newCode() (string, error) {
	buf := make([]byte, codeDigits)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		buf[i] = byte('0' + n.Int64())
	}
	return string(buf), nil
}

// hashCode returns the SHA-256 of a plaintext code. Only this hash is stored,
// matching the session token model: a database read cannot recover the code
// that was delivered out of band.
func hashCode(code string) []byte {
	sum := sha256.Sum256([]byte(code))
	return sum[:]
}

// isNoRows reports whether err is pgx's no-rows sentinel, the signal the
// service reads as "absent" rather than a failure.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// constantTimeEqual compares two hashes without leaking their contents through
// timing. The hashes are fixed length, so this is defensive rather than
// strictly required, but it keeps the code-check path uniform.
func constantTimeEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}
