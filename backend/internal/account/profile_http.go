package account

import (
	"encoding/json"
	"net/http"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// reputationBody is the Reputation schema. In US1 no reviews exist yet, so every
// profile reports enough_data false with a zero review_count and null rates. The
// shape is fixed now so the contract field is populated; the reputation package
// fills it later.
type reputationBody struct {
	EnoughData     bool     `json:"enough_data"`
	CompletionRate *int     `json:"completion_rate"`
	AverageRating  *float64 `json:"average_rating"`
	ReviewCount    int      `json:"review_count"`
}

// emptyReputation is the reputation of a profile with no reviews: not enough
// data, nothing to average, zero reviews.
func emptyReputation() reputationBody {
	return reputationBody{EnoughData: false, ReviewCount: 0}
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

// handleGetProfileMe returns the caller's own profile. The profile is born with
// the account, so a missing row is an invariant violation, not a 404.
func (s *Service) handleGetProfileMe(w http.ResponseWriter, r *http.Request, acc sqlcgen.UserAccount) {
	row, err := s.getMyProfile(r.Context(), acc.ID)
	if err != nil {
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
		VerificationStatus: nil,
		Reputation:         emptyReputation(),
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
		VerificationStatus: nil,
		Reputation:         emptyReputation(),
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
		Reputation:       emptyReputation(),
	})
}
