package account

import (
	"context"
	"testing"
)

// TestCreateAdmin_Idempotent_FR009 runs the admin creation path twice with the
// same email and asserts a single admin row results, the second call resetting
// the password rather than duplicating the account. admin:create must be safe to
// re-run (tasks.md T020), and the has_at_least_one_role/admin_has_no_business_role
// constraints only accept an admin whose two business roles are false.
func TestCreateAdmin_Idempotent_FR009(t *testing.T) {
	h := newHarness(t, "admin_create")
	ctx := context.Background()

	first, err := h.svc.CreateAdmin(ctx, "admin@contoh.test", "628123456789", "sandi-pertama")
	if err != nil {
		t.Fatalf("CreateAdmin pertama: %v", err)
	}
	if !first.RoleAdmin || first.RoleSubcontractor || first.RoleBuyer {
		t.Errorf("peran admin = admin:%v sub:%v buyer:%v, mau admin saja",
			first.RoleAdmin, first.RoleSubcontractor, first.RoleBuyer)
	}
	if !passwordMatches(first.PasswordHash, "sandi-pertama") {
		t.Error("kata sandi pertama tidak cocok dengan hash tersimpan")
	}

	second, err := h.svc.CreateAdmin(ctx, "admin@contoh.test", "628123456789", "sandi-kedua")
	if err != nil {
		t.Fatalf("CreateAdmin kedua: %v", err)
	}
	if uuidString(second.ID) != uuidString(first.ID) {
		t.Errorf("id admin berubah: %s -> %s", uuidString(first.ID), uuidString(second.ID))
	}
	if !passwordMatches(second.PasswordHash, "sandi-kedua") {
		t.Error("kata sandi kedua tidak menggantikan yang lama")
	}
	if passwordMatches(second.PasswordHash, "sandi-pertama") {
		t.Error("kata sandi lama masih berlaku setelah reset")
	}

	var count int64
	if err := h.pool.QueryRow(ctx,
		"SELECT count(*)::bigint FROM user_account WHERE role_admin").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("baris admin = %d, mau 1", count)
	}
}
