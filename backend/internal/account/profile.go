package account

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// Indonesia's coordinate bounds, copied from the coordinates_within_indonesia
// constraint on business_profile. Validating in Go turns an out-of-range pair
// into a 422 the owner can act on, instead of a constraint-violation 500.
const (
	minLatitude  = -11.5
	maxLatitude  = 6.5
	minLongitude = 94.5
	maxLongitude = 141.5
)

// profileInput is the validated ProfileUpdateRequest for PUT /profile/me. City
// and business name are always present; the coordinate pair is optional but
// must arrive complete or empty, never half, to satisfy
// coordinates_complete_or_empty. Description is free text, empty means clear it.
type profileInput struct {
	BusinessName string
	CityCode     string
	Description  string
	Latitude     *float64
	Longitude    *float64
}

// validateProfileInput checks the fields the owner may change and returns the
// per-field errors the handler renders as a 422. The rules mirror the table
// constraints so a write never reaches Postgres only to bounce: business name at
// least three characters after trim, coordinates complete or empty as a pair,
// and, when present, within Indonesia's bounds.
func validateProfileInput(in profileInput) []httpx.FieldError {
	var fields []httpx.FieldError
	if len(strings.TrimSpace(in.BusinessName)) < 3 {
		fields = append(fields, httpx.FieldError{Field: "business_name", Message: "Nama usaha minimal 3 karakter."})
	}
	if !cityCodeRe.MatchString(in.CityCode) {
		fields = append(fields, httpx.FieldError{Field: "city_code", Message: "Kode kota harus empat digit angka."})
	}
	// A coordinate is meaningful only as a lat/long pair. One without the other is
	// rejected here so the profile never carries a half-set location.
	if (in.Latitude == nil) != (in.Longitude == nil) {
		if in.Latitude == nil {
			fields = append(fields, httpx.FieldError{Field: "latitude", Message: "Lintang dan bujur harus diisi bersamaan."})
		} else {
			fields = append(fields, httpx.FieldError{Field: "longitude", Message: "Lintang dan bujur harus diisi bersamaan."})
		}
	}
	if in.Latitude != nil && (*in.Latitude < minLatitude || *in.Latitude > maxLatitude) {
		fields = append(fields, httpx.FieldError{Field: "latitude", Message: "Lintang di luar wilayah Indonesia."})
	}
	if in.Longitude != nil && (*in.Longitude < minLongitude || *in.Longitude > maxLongitude) {
		fields = append(fields, httpx.FieldError{Field: "longitude", Message: "Bujur di luar wilayah Indonesia."})
	}
	return fields
}

// getMyProfile loads the caller's own profile by account id. The profile is born
// with the account in register, so this never 404s for a real account; a missing
// row is a genuine invariant violation the handler turns into a 500.
func (s *Service) getMyProfile(ctx context.Context, accountID pgtype.UUID) (sqlcgen.GetProfileByAccountRow, error) {
	return s.queries().GetProfileByAccount(ctx, accountID)
}

// updateProfile writes the owner-editable fields. City existence is checked
// first so an unknown code answers errCityUnknown (422) rather than a foreign
// key 500. The coordinate pair is written as NULL when absent.
func (s *Service) updateProfile(ctx context.Context, accountID pgtype.UUID, in profileInput) (sqlcgen.GetProfileByAccountRow, error) {
	cityExists, err := s.queries().CityExists(ctx, in.CityCode)
	if err != nil {
		return sqlcgen.GetProfileByAccountRow{}, err
	}
	if !cityExists {
		return sqlcgen.GetProfileByAccountRow{}, errCityUnknown
	}
	current, err := s.queries().GetProfileByAccount(ctx, accountID)
	if err != nil {
		return sqlcgen.GetProfileByAccountRow{}, err
	}
	lat, err := numericFromPtr(in.Latitude)
	if err != nil {
		return sqlcgen.GetProfileByAccountRow{}, err
	}
	long, err := numericFromPtr(in.Longitude)
	if err != nil {
		return sqlcgen.GetProfileByAccountRow{}, err
	}
	if _, err := s.queries().UpdateProfile(ctx, sqlcgen.UpdateProfileParams{
		ID:           current.ID,
		BusinessName: strings.TrimSpace(in.BusinessName),
		CityCode:     in.CityCode,
		Latitude:     lat,
		Longitude:    long,
		Description:  textFromString(in.Description),
		UpdatedAt:    tstz(s.clock.Now()),
	}); err != nil {
		return sqlcgen.GetProfileByAccountRow{}, err
	}
	// Re-read through the joined query so the response carries the city and
	// province names the contract exposes as read-only fields.
	return s.queries().GetProfileByAccount(ctx, accountID)
}

// publicProfile loads any profile by its id for the public view. A missing or
// malformed id is errAccountUnknown so the handler answers 404 without leaking
// whether the id was well formed.
func (s *Service) publicProfile(ctx context.Context, id pgtype.UUID) (sqlcgen.GetProfileByIDRow, error) {
	row, err := s.queries().GetProfileByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return sqlcgen.GetProfileByIDRow{}, errAccountUnknown
		}
		return sqlcgen.GetProfileByIDRow{}, err
	}
	return row, nil
}

// numericFromPtr converts an optional float64 into a pgtype.Numeric, NULL when
// the pointer is nil. pgtype.Numeric has no float setter, so the value is
// formatted and scanned: the numeric(9,6) column keeps the six-decimal scale.
func numericFromPtr(f *float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if f == nil {
		return n, nil
	}
	if err := n.Scan(strconv.FormatFloat(*f, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}

// floatFromNumeric renders a pgtype.Numeric as an optional float64: an invalid
// (NULL) value becomes nil so the coordinate serializes to null.
func floatFromNumeric(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	v, err := n.Float64Value()
	if err != nil || !v.Valid {
		return nil
	}
	f := v.Float64
	return &f
}

// textFromString maps a string to a pgtype.Text, storing NULL for empty input so
// a cleared description is a null column rather than an empty string.
func textFromString(s string) pgtype.Text {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// textPtr maps an optional column back to a nullable string for the response.
func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	v := t.String
	return &v
}

// parseUUID parses a canonical uuid string into a pgtype.UUID. It reports ok
// false on any unparseable input, which the public profile handler turns into a
// 404 rather than a 500: a malformed id in the path names no profile.
func parseUUID(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, false
	}
	return u, u.Valid
}
