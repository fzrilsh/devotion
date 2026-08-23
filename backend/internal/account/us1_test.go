package account

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestPatchRoles_MenambahPeran_Berhasil_FR001 proves an account registered with
// one role can add another later without creating a new account: a buyer-only
// account gains the subcontractor role through PATCH /me/roles and reads it back
// on the returned MyAccount. FR-001.
func TestPatchRoles_MenambahPeran_Berhasil_FR001(t *testing.T) {
	h := newHarness(t, "roles_add_ok")
	cookie := h.registerAndLogin(t, "peran@example.com", "+6281300010001", "rahasia123")

	rec := h.do("PATCH", "/api/me/roles", map[string]any{
		"subcontractor": true,
		"buyer":         true,
	}, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200, body %s", rec.Code, rec.Body.String())
	}
	var me myAccount
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !me.Roles.Subcontractor || !me.Roles.Buyer {
		t.Fatalf("roles = %+v, mau kedua peran aktif", me.Roles)
	}
}

// TestPatchRoles_TanpaSesi_Unauthorized_FR001 proves changing roles requires a
// session: the route is gated, so an anonymous caller is 401. FR-001.
func TestPatchRoles_TanpaSesi_Unauthorized_FR001(t *testing.T) {
	h := newHarness(t, "roles_noauth")
	rec := h.do("PATCH", "/api/me/roles", map[string]any{
		"subcontractor": true,
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, mau 401", rec.Code)
	}
}

// TestPatchRoles_MencabutSemuaPeran_Ditolak_FR001 proves an account cannot drop
// to zero roles: clearing both flags is a 422 rather than a
// has_at_least_one_role constraint 500. FR-001.
func TestPatchRoles_MencabutSemuaPeran_Ditolak_FR001(t *testing.T) {
	h := newHarness(t, "roles_none")
	cookie := h.registerAndLogin(t, "kosong@example.com", "+6281300020002", "rahasia123")

	rec := h.do("PATCH", "/api/me/roles", map[string]any{
		"subcontractor": false,
		"buyer":         false,
	}, cookie)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422, body %s", rec.Code, rec.Body.String())
	}
}

// TestVerifyPhone_JalurBerhasil_FR002 proves the phone one-time code drives
// verify-phone to mark the number verified, the phone half of the FR-002 gate
// that both channels must clear before an account can publish a listing or send
// a quota request. FR-002.
func TestVerifyPhone_JalurBerhasil_FR002(t *testing.T) {
	h := newHarness(t, "verify_phone_ok")
	cookie := h.registerAndLogin(t, "hp@example.com", "+6281300030003", "rahasia123")

	code := h.delivery.phone
	if !codeRe.MatchString(code) {
		t.Fatalf("kode HP = %q, mau enam digit", code)
	}
	rec := h.do("POST", "/api/auth/verify-phone", map[string]any{"code": code}, cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("verify-phone: status %d, body %s", rec.Code, rec.Body.String())
	}

	rec = h.do("GET", "/api/me", nil, cookie)
	var me myAccount
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !me.PhoneVerified {
		t.Fatal("phone_verified mau true setelah verifikasi")
	}
}

// TestVerifyPhone_KodeSalah_Ditolak_FR002 proves a wrong code does not verify
// the number: verify-phone answers 422 and the account stays unverified. FR-002.
func TestVerifyPhone_KodeSalah_Ditolak_FR002(t *testing.T) {
	h := newHarness(t, "verify_phone_wrong")
	cookie := h.registerAndLogin(t, "hpsalah@example.com", "+6281300040004", "rahasia123")

	rec := h.do("POST", "/api/auth/verify-phone", map[string]any{"code": "000000"}, cookie)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422, body %s", rec.Code, rec.Body.String())
	}

	rec = h.do("GET", "/api/me", nil, cookie)
	var me myAccount
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if me.PhoneVerified {
		t.Fatal("phone_verified mau tetap false setelah kode salah")
	}
}

// TestResendCode_ChannelTidakSah_Ditolak_FR002 proves resend-code refuses a
// request that names neither email nor whatsapp: the input is a 422 before any
// code is issued. FR-002.
func TestResendCode_ChannelTidakSah_Ditolak_FR002(t *testing.T) {
	h := newHarness(t, "resend_bad_channel")
	rec := h.do("POST", "/api/auth/resend-code", map[string]any{
		"target": "resend@example.com", "channel": "sms",
	}, "")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422, body %s", rec.Code, rec.Body.String())
	}
}

// TestResendCode_SelaluDiterima_FR002 proves resend-code answers 202 whether or
// not the target exists, so it re-issues a verification code without revealing
// account existence. FR-002.
func TestResendCode_SelaluDiterima_FR002(t *testing.T) {
	h := newHarness(t, "resend_ok")
	h.registerAndLogin(t, "resendada@example.com", "+6281300050005", "rahasia123")

	for _, target := range []string{"resendada@example.com", "tidakada@example.com"} {
		rec := h.do("POST", "/api/auth/resend-code", map[string]any{
			"target": target, "channel": "email",
		}, "")
		if rec.Code != http.StatusAccepted {
			t.Fatalf("target %q: status %d, mau 202, body %s", target, rec.Code, rec.Body.String())
		}
	}
}

// TestPublicProfile_IdTidakDikenal_NotFound_FR016 proves an unknown but
// well-formed profile id is a 404 that reveals nothing, matching the malformed
// id path, so the public route never confirms which ids exist. FR-016.
func TestPublicProfile_IdTidakDikenal_NotFound_FR016(t *testing.T) {
	h := newHarness(t, "public_notfound")
	rec := h.do("GET", "/api/profile/00000000-0000-0000-0000-000000000000", nil, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, mau 404, body %s", rec.Code, rec.Body.String())
	}
}
