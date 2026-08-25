package account

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// profileView mirrors the fields of myProfileBody and publicProfileBody a test
// needs to read back. It is a loose superset so one struct decodes either body.
type profileView struct {
	ProfileID        string   `json:"profile_id"`
	BusinessName     string   `json:"business_name"`
	Description      *string  `json:"description"`
	CityCode         *string  `json:"city_code"`
	CityName         *string  `json:"city_name"`
	ProvinceCode     *string  `json:"province_code"`
	ProvinceName     *string  `json:"province_name"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	IdentityVerified bool     `json:"identity_verified"`
}

// TestRegister_MembuatProfilDalamSatuTransaksi_FR004 proves registration lands
// both the account and its business profile, so GET /profile/me answers 200 for
// a freshly registered account instead of 404. FR-004.
func TestRegister_MembuatProfilDalamSatuTransaksi_FR004(t *testing.T) {
	h := newHarness(t, "profile_reg_tx")
	cookie := h.registerAndLogin(t, "profil@example.com", "+6281300001111", "rahasia123")

	rec := h.do("GET", "/api/profile/me", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /profile/me: status %d, mau 200, body %s", rec.Code, rec.Body.String())
	}
	var pv profileView
	if err := json.Unmarshal(rec.Body.Bytes(), &pv); err != nil {
		t.Fatalf("decode profil: %v", err)
	}
	if pv.ProfileID == "" {
		t.Fatal("profile_id kosong, mau terisi karena profil lahir bersama akun")
	}
	if pv.BusinessName != "Konveksi Contoh" {
		t.Fatalf("business_name = %q, mau %q", pv.BusinessName, "Konveksi Contoh")
	}

	// A profile row must exist for this account.
	var count int
	if err := h.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM business_profile").Scan(&count); err != nil {
		t.Fatalf("query profil: %v", err)
	}
	if count != 1 {
		t.Fatalf("baris business_profile = %d, mau 1", count)
	}
}

// TestRegister_KotaTidakDikenal_Ditolak_FR004 proves an unknown but well-formed
// city code is a 422 naming city_code, not a foreign key 500, and leaves no
// account behind. FR-004.
func TestRegister_KotaTidakDikenal_Ditolak_FR004(t *testing.T) {
	h := newHarness(t, "profile_reg_kota")
	rec := h.do("POST", "/api/auth/register", map[string]any{
		"email": "kota@example.com", "phone": "+6281300002222", "password": "rahasia123",
		"business_name": "Konveksi Contoh",
		"city_code":     "9999", // four digits, but no such city seeded
		"roles":         map[string]any{"buyer": true},
	}, "")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422, body %s", rec.Code, rec.Body.String())
	}

	var count int
	if err := h.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM user_account WHERE email = $1", "kota@example.com").Scan(&count); err != nil {
		t.Fatalf("query akun: %v", err)
	}
	if count != 0 {
		t.Fatal("akun tersisa padahal kota tidak dikenal")
	}
}

// TestRegister_GagalMembuatProfil_TidakMenyisakanAkun_FR004 proves the account
// and profile share one transaction: when the profile insert fails, the account
// it would have belonged to is rolled back too. It drives the service directly
// with a business name the DB check rejects (the handler would have caught it,
// so this reaches past the handler to the transaction itself). FR-004.
func TestRegister_GagalMembuatProfil_TidakMenyisakanAkun_FR004(t *testing.T) {
	h := newHarness(t, "profile_reg_rollback")

	_, err := h.svc.register(context.Background(), registerInput{
		Email:        "rollback@example.com",
		Phone:        "6281300003333",
		Password:     "rahasia123",
		BusinessName: "ab", // len 2, violates business_name_not_empty CHECK
		CityCode:     testCityCode,
		Buyer:        true,
	})
	if err == nil {
		t.Fatal("register mau gagal karena nama usaha melanggar constraint")
	}

	var count int
	if err := h.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM user_account WHERE email = $1", "rollback@example.com").Scan(&count); err != nil {
		t.Fatalf("query akun: %v", err)
	}
	if count != 0 {
		t.Fatal("akun tersisa padahal pembuatan profil gagal, transaksi tidak di-rollback")
	}
}

// TestGetMyProfile_MenurunkanProvinsiDariKota_FR004 proves the profile view
// resolves city and province names from the city join rather than storing them,
// so the client gets human-readable location without a second lookup. FR-004.
func TestGetMyProfile_MenurunkanProvinsiDariKota_FR004(t *testing.T) {
	h := newHarness(t, "profile_me_provinsi")
	cookie := h.registerAndLogin(t, "provinsi@example.com", "+6281300004444", "rahasia123")

	rec := h.do("GET", "/api/profile/me", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200", rec.Code)
	}
	var pv profileView
	if err := json.Unmarshal(rec.Body.Bytes(), &pv); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pv.CityName == nil || *pv.CityName != "Kota Bandung" {
		t.Fatalf("city_name = %v, mau %q", pv.CityName, "Kota Bandung")
	}
	if pv.ProvinceCode == nil || *pv.ProvinceCode != "32" {
		t.Fatalf("province_code = %v, mau %q", pv.ProvinceCode, "32")
	}
	if pv.ProvinceName == nil || *pv.ProvinceName != "Jawa Barat" {
		t.Fatalf("province_name = %v, mau %q", pv.ProvinceName, "Jawa Barat")
	}
}

// TestUpdateProfile_MengubahKoordinatSendiri_FR057 proves the owner can set the
// business name, description, and a valid coordinate pair through PUT
// /profile/me, and reads them back. FR-057.
func TestUpdateProfile_MengubahKoordinatSendiri_FR057(t *testing.T) {
	h := newHarness(t, "profile_update_ok")
	cookie := h.registerAndLogin(t, "koordinat@example.com", "+6281300005555", "rahasia123")

	rec := h.do("PUT", "/api/profile/me", map[string]any{
		"business_name": "Konveksi Baru",
		"description":   "Spesialis kaos",
		"city_code":     testCityCode,
		"latitude":      -6.914744,
		"longitude":     107.609810,
	}, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200, body %s", rec.Code, rec.Body.String())
	}
	var pv profileView
	if err := json.Unmarshal(rec.Body.Bytes(), &pv); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pv.BusinessName != "Konveksi Baru" {
		t.Fatalf("business_name = %q", pv.BusinessName)
	}
	if pv.Description == nil || *pv.Description != "Spesialis kaos" {
		t.Fatalf("description = %v", pv.Description)
	}
	if pv.Latitude == nil || pv.Longitude == nil {
		t.Fatal("koordinat mau terisi setelah update")
	}
}

// TestUpdateProfile_KoordinatDiLuarIndonesia_Ditolak_FR057 proves a coordinate
// pair outside Indonesia's bounds is a 422 rather than a constraint-violation
// 500. FR-057.
func TestUpdateProfile_KoordinatDiLuarIndonesia_Ditolak_FR057(t *testing.T) {
	h := newHarness(t, "profile_update_luar")
	cookie := h.registerAndLogin(t, "luar@example.com", "+6281300006666", "rahasia123")

	rec := h.do("PUT", "/api/profile/me", map[string]any{
		"business_name": "Konveksi Contoh",
		"city_code":     testCityCode,
		"latitude":      48.8566, // Paris, well outside Indonesia
		"longitude":     2.3522,
	}, cookie)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422", rec.Code)
	}
}

// TestUpdateProfile_LintangTanpaBujur_Ditolak_FR057 proves a half-set location
// is rejected as a 422: a coordinate is meaningful only as a lat/long pair.
// FR-057.
func TestUpdateProfile_LintangTanpaBujur_Ditolak_FR057(t *testing.T) {
	h := newHarness(t, "profile_update_setengah")
	cookie := h.registerAndLogin(t, "setengah@example.com", "+6281300007777", "rahasia123")

	rec := h.do("PUT", "/api/profile/me", map[string]any{
		"business_name": "Konveksi Contoh",
		"city_code":     testCityCode,
		"latitude":      -6.914744,
	}, cookie)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422", rec.Code)
	}
}

// TestUpdateProfile_TanpaSesi_Unauthorized_FR004 proves editing a profile
// requires a session: the route is gated. FR-004.
func TestUpdateProfile_TanpaSesi_Unauthorized_FR004(t *testing.T) {
	h := newHarness(t, "profile_update_noauth")
	rec := h.do("PUT", "/api/profile/me", map[string]any{
		"business_name": "Konveksi Contoh",
		"city_code":     testCityCode,
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, mau 401", rec.Code)
	}
}

// TestGetProfileMe_AdminDitolak_FR005 proves an admin account is refused the
// business profile view with 403, not a 500 from an ErrNoRows the handler never
// mapped. Both profile routes are gated to the business roles, so an admin (which
// holds neither, per admin_has_no_business_role) is turned away at the router
// before the handler runs, the same way every other business endpoint refuses
// it. FR-005.
func TestGetProfileMe_AdminDitolak_FR005(t *testing.T) {
	h := newHarness(t, "profile_me_admin")
	cookie := h.createAdminAndLogin(t, "admin@example.com", "+6281300009999", "rahasia123")

	rec := h.do("GET", "/api/profile/me", nil, cookie)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403, body %s", rec.Code, rec.Body.String())
	}
	var prob struct {
		Code   string `json:"code"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &prob); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if prob.Code != "FORBIDDEN" {
		t.Fatalf("code = %q, mau FORBIDDEN", prob.Code)
	}
}

// TestGetProfileMe_AkunUsahaBerhasil_FR004 proves a business account still gets
// its profile with 200, so the admin gate does not shut the door on the accounts
// that do own a profile. FR-004.
func TestGetProfileMe_AkunUsahaBerhasil_FR004(t *testing.T) {
	h := newHarness(t, "profile_me_usaha")
	cookie := h.registerAndLogin(t, "usaha@example.com", "+6281300001010", "rahasia123")

	rec := h.do("GET", "/api/profile/me", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200, body %s", rec.Code, rec.Body.String())
	}
	var pv profileView
	if err := json.Unmarshal(rec.Body.Bytes(), &pv); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pv.ProfileID == "" {
		t.Fatal("profile_id kosong untuk akun usaha")
	}
}

// TestPutProfileMe_AdminDitolak_FR005 proves an admin account is refused the
// profile update with 403, not a 500. PUT shares the business-role gate with
// GET, so the admin is turned away at the router before the handler runs. This
// is the same class of bug the GET gate fixed: without a role gate PUT would
// reach updateProfile, read no profile row, and 500. FR-005.
func TestPutProfileMe_AdminDitolak_FR005(t *testing.T) {
	h := newHarness(t, "profile_put_admin")
	cookie := h.createAdminAndLogin(t, "admin@example.com", "+6281300009999", "rahasia123")

	rec := h.do("PUT", "/api/profile/me", map[string]any{
		"business_name": "Konveksi Contoh",
		"city_code":     testCityCode,
	}, cookie)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403, body %s", rec.Code, rec.Body.String())
	}
	var prob struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &prob); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if prob.Code != "FORBIDDEN" {
		t.Fatalf("code = %q, mau FORBIDDEN", prob.Code)
	}
}

// TestPutProfileMe_AkunUsahaBerhasil_FR057 proves a business account still
// updates its profile with 200, so the business-role gate on PUT does not shut
// the door on the accounts that own a profile. FR-057.
func TestPutProfileMe_AkunUsahaBerhasil_FR057(t *testing.T) {
	h := newHarness(t, "profile_put_usaha")
	cookie := h.registerAndLogin(t, "usaha@example.com", "+6281300001010", "rahasia123")

	rec := h.do("PUT", "/api/profile/me", map[string]any{
		"business_name": "Konveksi Baru",
		"city_code":     testCityCode,
	}, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200, body %s", rec.Code, rec.Body.String())
	}
	var pv profileView
	if err := json.Unmarshal(rec.Body.Bytes(), &pv); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pv.BusinessName != "Konveksi Baru" {
		t.Fatalf("business_name = %q", pv.BusinessName)
	}
}

// reachable without a session, so any visitor can see a subcontractor's public
// details. FR-016.
func TestPublicProfile_TanpaSesi_Berhasil_FR016(t *testing.T) {
	h := newHarness(t, "profile_public")
	cookie := h.registerAndLogin(t, "public@example.com", "+6281300008888", "rahasia123")

	// Read the owner's profile id from the authenticated view, then fetch it
	// anonymously through the public route.
	rec := h.do("GET", "/api/profile/me", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /profile/me: status %d", rec.Code)
	}
	var mine profileView
	if err := json.Unmarshal(rec.Body.Bytes(), &mine); err != nil {
		t.Fatalf("decode: %v", err)
	}

	rec = h.do("GET", "/api/profile/"+mine.ProfileID, nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /profile/{id} tanpa sesi: status %d, mau 200", rec.Code)
	}
	var pub profileView
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatalf("decode publik: %v", err)
	}
	if pub.ProfileID != mine.ProfileID {
		t.Fatalf("profile_id publik = %q, mau %q", pub.ProfileID, mine.ProfileID)
	}
}
