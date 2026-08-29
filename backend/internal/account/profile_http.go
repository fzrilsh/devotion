package account

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
	"github.com/fzrilsh/devotion/backend/internal/reputation"
)

// reputationBody is the Reputation schema, owned by the reputation package so
// the profile and a search result render one shape with one threshold (FR-073).
type reputationBody = reputation.View

// reputationOf computes a profile's read-time reputation (FR-071). It runs the
// same SearchReputation query the search page runs, over a one-element id array,
// and hands its scalars to reputation.Derive. Reusing that query rather than
// writing a second one is what keeps the profile and the search from reporting
// different completion rates for the same business: the FR-072 divisor rule
// exists in exactly one statement.
//
// A query failure degrades to the empty block rather than failing the profile
// read: reputation is informative, and a profile page that 500s over it would be
// worse than one that shows "belum cukup data".
func (s *Service) reputationOf(ctx context.Context, profileID pgtype.UUID) reputationBody {
	rows, err := s.queries().SearchReputation(ctx, []pgtype.UUID{profileID})
	if err != nil || len(rows) == 0 {
		return reputation.Derive(0, 0, 0, nil)
	}
	r := rows[0]
	return reputation.Derive(
		int(r.CompletionCompleted),
		int(r.CompletionDivisor),
		int(r.ReviewCount),
		floatFromNumeric(r.AverageRating),
	)
}

// myProfileBody is the MyProfile response. city_name, province_code, and
// province_name are read-only, resolved from the city join. identity_verified
// maps to business_profile.verified. verification_status is null in US1 because
// the verification_request table is still empty.
type myProfileBody struct {
	ProfileID          string         `json:"profile_id"`
	BusinessName       string         `json:"business_name"`
	Description        *string        `json:"description"`
	CityCode           *string        `json:"city_code"`
	CityName           *string        `json:"city_name"`
	ProvinceCode       *string        `json:"province_code"`
	ProvinceName       *string        `json:"province_name"`
	Latitude           *float64       `json:"latitude"`
	Longitude          *float64       `json:"longitude"`
	IdentityVerified   bool           `json:"identity_verified"`
	VerificationStatus *string        `json:"verification_status"`
	Reputation         reputationBody `json:"reputation"`
}

// publicProfileBody is the PublicProfile response. It hides the account's own
// coordinates? No: FR-016 exposes city, province, and coordinates for the map.
// listing is null in this handler; the listing package supplies it when a public
// profile view is joined with its listing.
type publicProfileBody struct {
	ProfileID        string         `json:"profile_id"`
	BusinessName     string         `json:"business_name"`
	Description      *string        `json:"description"`
	CityCode         *string        `json:"city_code"`
	CityName         *string        `json:"city_name"`
	ProvinceCode     *string        `json:"province_code"`
	ProvinceName     *string        `json:"province_name"`
	Latitude         *float64       `json:"latitude"`
	Longitude        *float64       `json:"longitude"`
	IdentityVerified bool           `json:"identity_verified"`
	Listing          json.RawMessage `json:"listing"`
	Reputation       reputationBody `json:"reputation"`
}

// strPtr returns a pointer to a copy of s, for the nullable string fields whose
// column is NOT NULL but the contract models as nullable.
func strPtr(s string) *string { return &s }

// verificationStatusOf returns the status of the profile's most recent
// verification submission, or nil when the profile never submitted one. A query
// error other than no-rows is logged and treated as nil: the profile still
// renders, only the status badge is absent, which never blocks the page.
func (s *Service) verificationStatusOf(ctx context.Context, profileID pgtype.UUID) *string {
	st, err := s.queries().LatestVerificationStatusByProfile(ctx, profileID)
	if err != nil {
		if !isNoRows(err) {
			slog.ErrorContext(ctx, "account: gagal membaca status verifikasi profil",
				"profile_id", uuidString(profileID))
		}
		return nil
	}
	v := string(st)
	return &v
}

// handleGetProfileMe returns the caller's own profile. The route is gated to the
// business roles, so an admin never reaches this handler (it is refused with 403
// at the router, the same as every other business endpoint). For a business
// account the profile is born with it in one transaction (FR-004), so a missing
// row here is an invariant break, not a 404: it is logged with the account id and
// answered 500, because if it ever happens it means the data is corrupt, not that
// the caller did anything wrong.
func (s *Service) handleGetProfileMe(w http.ResponseWriter, r *http.Request, acc sqlcgen.UserAccount) {
	row, err := s.getMyProfile(r.Context(), acc.ID)
	if err != nil {
		if isNoRows(err) {
			slog.ErrorContext(r.Context(), "account: akun usaha tanpa baris profil, invarian rusak",
				"account_id", uuidString(acc.ID))
		}
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, myProfileBody{
		ProfileID:          uuidString(row.ID),
		BusinessName:       row.BusinessName,
		Description:        textPtr(row.Description),
		CityCode:           strPtr(row.CityCode),
		CityName:           strPtr(row.CityName),
		ProvinceCode:       strPtr(row.ProvinceCode),
		ProvinceName:       strPtr(row.ProvinceName),
		Latitude:           floatFromNumeric(row.Latitude),
		Longitude:          floatFromNumeric(row.Longitude),
		IdentityVerified:   row.Verified,
		VerificationStatus: s.verificationStatusOf(r.Context(), row.ID),
		Reputation:         s.reputationOf(r.Context(), row.ID),
	})
}

// handlePutProfileMe writes the owner-editable fields of the caller's profile
// and returns the updated view. Invalid input is a 422 naming the field; an
// unknown city is a 422 on city_code, not a foreign key 500.
func (s *Service) handlePutProfileMe(w http.ResponseWriter, r *http.Request, acc sqlcgen.UserAccount) {
	var body struct {
		BusinessName string   `json:"business_name"`
		Description  string   `json:"description"`
		CityCode     string   `json:"city_code"`
		Latitude     *float64 `json:"latitude"`
		Longitude    *float64 `json:"longitude"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	in := profileInput{
		BusinessName: body.BusinessName,
		CityCode:     body.CityCode,
		Description:  body.Description,
		Latitude:     body.Latitude,
		Longitude:    body.Longitude,
	}
	if fields := validateProfileInput(in); len(fields) > 0 {
		httpx.WriteValidation(w, "Masukan tidak sah.", fields)
		return
	}
	row, err := s.updateProfile(r.Context(), acc.ID, in)
	if err != nil {
		if err == errCityUnknown {
			httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
				{Field: "city_code", Message: "Kota tidak dikenal."},
			})
			return
		}
		if isNoRows(err) {
			slog.ErrorContext(r.Context(), "account: akun usaha tanpa baris profil, invarian rusak",
				"account_id", uuidString(acc.ID))
		}
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, myProfileBody{
		ProfileID:          uuidString(row.ID),
		BusinessName:       row.BusinessName,
		Description:        textPtr(row.Description),
		CityCode:           strPtr(row.CityCode),
		CityName:           strPtr(row.CityName),
		ProvinceCode:       strPtr(row.ProvinceCode),
		ProvinceName:       strPtr(row.ProvinceName),
		Latitude:           floatFromNumeric(row.Latitude),
		Longitude:          floatFromNumeric(row.Longitude),
		IdentityVerified:   row.Verified,
		VerificationStatus: s.verificationStatusOf(r.Context(), row.ID),
		Reputation:         s.reputationOf(r.Context(), row.ID),
	})
}

// handleGetPublicProfile returns any profile by its id for the public view. A
// malformed or unknown id is a 404 that reveals nothing about which it was.
func (s *Service) handleGetPublicProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(r.PathValue("profileId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Profil tidak ditemukan.")
		return
	}
	row, err := s.publicProfile(r.Context(), id)
	if err != nil {
		if err == errAccountUnknown {
			httpx.WriteProblem(w, httpx.CodeNotFound, "Profil tidak ditemukan.")
			return
		}
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, publicProfileBody{
		ProfileID:        uuidString(row.ID),
		BusinessName:     row.BusinessName,
		Description:      textPtr(row.Description),
		CityCode:         strPtr(row.CityCode),
		CityName:         strPtr(row.CityName),
		ProvinceCode:     strPtr(row.ProvinceCode),
		ProvinceName:     strPtr(row.ProvinceName),
		Latitude:         floatFromNumeric(row.Latitude),
		Longitude:        floatFromNumeric(row.Longitude),
		IdentityVerified: row.Verified,
		Listing:          nil,
		Reputation:       s.reputationOf(r.Context(), row.ID),
	})
}
