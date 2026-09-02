package account

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
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

// TestPatchRoles_MencabutPeranSaatAdaOrderAktif_Ditolak_FR001 proves a role
// cannot be revoked while an active order still depends on it: the account holds
// a work order as buyer, so dropping the buyer role is a 409 ROLES_IN_USE rather
// than a silent revoke that would strand the order. Setting subcontractor true
// keeps this off the both-false 422 path, isolating the in-use conflict. FR-001.
func TestPatchRoles_MencabutPeranSaatAdaOrderAktif_Ditolak_FR001(t *testing.T) {
	h := newHarness(t, "roles_in_use")
	cookie := h.registerAndLogin(t, "aktif@example.com", "+6281300060006", "rahasia123")

	ctx := context.Background()
	weekNow := platform.WeekStart(baseTime)
	deadline := weekNow.AddDate(0, 0, 7*4)

	// The caller registered buyer-only with a profile; its id is the buyer_id an
	// active order keys on, which is what CountActiveOrdersAsBuyer counts.
	var buyerProfile pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`SELECT bp.id FROM business_profile bp JOIN user_account ua ON ua.id = bp.account_id
		 WHERE ua.email = $1`, "aktif@example.com").Scan(&buyerProfile); err != nil {
		t.Fatalf("ambil profil pembeli: %v", err)
	}

	// A distinct subcontractor party satisfies two_distinct_parties and the
	// reject_self_request trigger; it owns the listing the order draws from.
	var subAcc, subProfile pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO user_account (email, phone, password_hash, role_subcontractor, created_at, updated_at)
		 VALUES ($1, $2, 'x', true, $3, $3) RETURNING id`,
		"subkon-aktif@example.com", "628300060007", baseTime).Scan(&subAcc); err != nil {
		t.Fatalf("seed subcontractor account: %v", err)
	}
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO business_profile (account_id, business_name, city_code, verified, created_at, updated_at)
		 VALUES ($1, 'Subkon Contoh', $2, true, $3, $3) RETURNING id`,
		subAcc, testCityCode, baseTime).Scan(&subProfile); err != nil {
		t.Fatalf("seed subcontractor profile: %v", err)
	}

	var productID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO catalog_item (type, name, active, created_at) VALUES ('product', 'Kaos Oblong', true, $1) RETURNING id`,
		baseTime).Scan(&productID); err != nil {
		t.Fatalf("seed catalog item: %v", err)
	}

	var listingID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO capacity_listing (profile_id, weekly_capacity, readiness_lead_days, published, calendar_updated_at, horizon_until, created_at, updated_at)
		 VALUES ($1, 500, 0, true, $2, $3, $2, $2) RETURNING id`,
		subProfile, baseTime, weekNow.AddDate(0, 0, 7*8)).Scan(&listingID); err != nil {
		t.Fatalf("seed listing: %v", err)
	}
	if _, err := h.pool.Exec(ctx,
		`INSERT INTO listing_product (listing_id, item_id) VALUES ($1, $2)`,
		listingID, productID); err != nil {
		t.Fatalf("seed listing_product: %v", err)
	}

	var requestID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO quota_request (buyer_id, product_item_id, quantity, material, deadline, reply_due_at, created_at)
		 VALUES ($1, $2, 100, 'Katun', $3, $4, $5) RETURNING id`,
		buyerProfile, productID, deadline, baseTime.Add(48*time.Hour), baseTime).Scan(&requestID); err != nil {
		t.Fatalf("seed quota_request: %v", err)
	}

	var candidateID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO request_candidate (request_id, listing_id, subcontractor_id, status, updated_at)
		 VALUES ($1, $2, $3, 'agreed', $4) RETURNING id`,
		requestID, listingID, subProfile, baseTime).Scan(&candidateID); err != nil {
		t.Fatalf("seed request_candidate: %v", err)
	}

	var offerID pgtype.UUID
	if err := h.pool.QueryRow(ctx,
		`INSERT INTO offer (candidate_id, sequence, proposed_by, total_price, readiness_lead_days, created_at)
		 VALUES ($1, 1, 'subcontractor', 100000, 0, $2) RETURNING id`,
		candidateID, baseTime).Scan(&offerID); err != nil {
		t.Fatalf("seed offer: %v", err)
	}

	if _, err := h.pool.Exec(ctx,
		`INSERT INTO work_order (candidate_id, offer_id, buyer_id, subcontractor_id, quantity, total_price,
		     deadline, readiness_week_start, status, created_at)
		 VALUES ($1, $2, $3, $4, 100, 100000, $5, $6, 'accepted', $7)`,
		candidateID, offerID, buyerProfile, subProfile, deadline, weekNow, baseTime); err != nil {
		t.Fatalf("seed work_order: %v", err)
	}

	rec := h.do("PATCH", "/api/me/roles", map[string]any{
		"subcontractor": true,
		"buyer":         false,
	}, cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d, mau 409, body %s", rec.Code, rec.Body.String())
	}
	var p struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem %q: %v", rec.Body.String(), err)
	}
	if p.Code != string(httpx.CodeRolesInUse) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeRolesInUse)
	}
}

// TestVerifyPhone_JalurBerhasil_FR002 proves the phone one-time code drives
// verify-phone to mark the number verified, the phone half of the FR-002 gate
// that both channels must clear before an account can publish a listing or send
// a quota request. FR-002.
func TestVerifyPhone_JalurBerhasil_FR002(t *testing.T) {
	h := newHarness(t, "verify_phone_ok")
	cookie := h.registerAndLogin(t, "hp@example.com", "+6281300030003", "rahasia123")

	code := h.delivery.waitPhone(t)
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
