package storage

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/db/testdb"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// baseTime is a fixed instant so any clock-derived created_at is deterministic.
var baseTime = time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)

// seedProfile inserts the province, city, account, and business_profile a file
// row must reference, and returns the new profile id. It writes SQL directly
// because storage has no need for account or profile queries of its own.
func seedProfile(t *testing.T, pool *pgxpool.Pool, email string) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO province (code, name) VALUES ('32', 'Jawa Barat') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seed province: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO city (code, province_code, name) VALUES ('3273', '32', 'Bandung') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seed city: %v", err)
	}
	var accountID pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO user_account (email, phone, password_hash, created_at, updated_at)
		 VALUES ($1, $2, 'x', $3, $3) RETURNING id`,
		email, "+62812"+email[:4], baseTime).Scan(&accountID); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	var profileID pgtype.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO business_profile (account_id, business_name, city_code, created_at, updated_at)
		 VALUES ($1, 'Konveksi Uji', '3273', $2, $2) RETURNING id`,
		accountID, baseTime).Scan(&profileID); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	return profileID
}

// jpegBytes returns a valid one-pixel JPEG so a test drives the real decode and
// re-encode path rather than a hand-built header.
func jpegBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func newService(t *testing.T, pool *pgxpool.Pool, fileMB, totalMB int) *Service {
	t.Helper()
	svc, err := New(pool, platform.NewTestClock(baseTime), t.TempDir(), fileMB, totalMB)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return svc
}

// TestOpen_OwnerAdminAndStranger_FR009 proves the access gate: the owner and an
// admin can open a stored file, but a stranger (a different profile, not admin)
// is refused with ErrForbidden and never receives a reader. This is the FR-009
// obligation that an identity document is reachable only by its owner and admin.
func TestOpen_OwnerAdminAndStranger_FR009(t *testing.T) {
	pool := testdb.New(t, "storage_access")
	svc := newService(t, pool, 5, 500)
	ctx := context.Background()

	owner := seedProfile(t, pool, "ownr@example.com")
	stranger := seedProfile(t, pool, "strg@example.com")

	saved, err := svc.Save(ctx, owner, sqlcgen.FileTypeIdentityDocument, "ktp.jpg", bytes.NewReader(jpegBytes(t)))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	id := saved.File.ID

	// Owner can open.
	rc, _, err := svc.Open(ctx, id, Caller{ProfileID: owner})
	if err != nil {
		t.Fatalf("owner Open: %v", err)
	}
	_ = rc.Close()

	// Admin can open even without a matching profile.
	rc, _, err = svc.Open(ctx, id, Caller{IsAdmin: true})
	if err != nil {
		t.Fatalf("admin Open: %v", err)
	}
	_ = rc.Close()

	// A stranger is forbidden and gets no reader.
	rc, _, err = svc.Open(ctx, id, Caller{ProfileID: stranger})
	if err != ErrForbidden {
		t.Fatalf("stranger Open err = %v, mau ErrForbidden", err)
	}
	if rc != nil {
		t.Fatal("stranger diberi reader, mau nil")
	}
}

// TestSave_DeceptiveExtension_Rejected_FR006 proves the type check reads the
// magic bytes, not the client's filename: a text body named .jpg is refused
// with ErrUnsupportedType and no row is written.
func TestSave_DeceptiveExtension_Rejected_FR006(t *testing.T) {
	pool := testdb.New(t, "storage_deceptive")
	svc := newService(t, pool, 5, 500)
	owner := seedProfile(t, pool, "dcpt@example.com")

	body := []byte("ini teks biasa, bukan gambar, meski namanya .jpg")
	_, err := svc.Save(context.Background(), owner, sqlcgen.FileTypeLocationPhoto, "foto.jpg", bytes.NewReader(body))
	if err != ErrUnsupportedType {
		t.Fatalf("Save err = %v, mau ErrUnsupportedType", err)
	}

	used, err := svc.queries().SumUploadedBytesByOwner(context.Background(), owner)
	if err != nil {
		t.Fatalf("sum: %v", err)
	}
	if used != 0 {
		t.Fatalf("byte tersimpan = %d setelah tolak, mau 0", used)
	}
}

// TestSave_QuotaFull_FR006 proves a file that would push the owner past the
// total quota is refused with ErrQuotaFull, the distinct error a handler maps
// to the clear "kuota penuh" message.
func TestSave_QuotaFull_FR006(t *testing.T) {
	pool := testdb.New(t, "storage_quota")
	// totalMB 0 makes the quota zero bytes, so any file overflows it.
	svc := newService(t, pool, 5, 0)
	owner := seedProfile(t, pool, "quot@example.com")

	_, err := svc.Save(context.Background(), owner, sqlcgen.FileTypeLocationPhoto, "foto.jpg", bytes.NewReader(jpegBytes(t)))
	if err != ErrQuotaFull {
		t.Fatalf("Save err = %v, mau ErrQuotaFull", err)
	}
}

// TestSave_TooLarge_FR006 proves a body over the per-file limit is refused with
// ErrTooLarge before it is decoded, so an oversized upload cannot exhaust memory.
func TestSave_TooLarge_FR006(t *testing.T) {
	pool := testdb.New(t, "storage_toolarge")
	// A 0MB per-file limit rejects any non-empty body without decoding it.
	svc := newService(t, pool, 0, 500)
	owner := seedProfile(t, pool, "larg@example.com")

	_, err := svc.Save(context.Background(), owner, sqlcgen.FileTypeLocationPhoto, "foto.jpg", bytes.NewReader(jpegBytes(t)))
	if err != ErrTooLarge {
		t.Fatalf("Save err = %v, mau ErrTooLarge", err)
	}
}

// TestSave_StripsExif_FR006 proves an image is re-encoded, not stored verbatim:
// a JPEG carrying an EXIF marker comes back out without it, which is how a
// location photo's GPS coordinates are dropped. The check reads the stored
// bytes back through Open and confirms the EXIF APP1 marker is gone.
func TestSave_StripsExif_FR006(t *testing.T) {
	pool := testdb.New(t, "storage_exif")
	svc := newService(t, pool, 5, 500)
	ctx := context.Background()
	owner := seedProfile(t, pool, "exif@example.com")

	// Splice a fake EXIF APP1 segment into a real JPEG after the SOI marker.
	src := jpegBytes(t)
	exif := []byte{0xFF, 0xE1, 0x00, 0x08, 'E', 'x', 'i', 'f', 0x00, 0x00}
	tampered := append(append(append([]byte{}, src[:2]...), exif...), src[2:]...)

	saved, err := svc.Save(ctx, owner, sqlcgen.FileTypeLocationPhoto, "lokasi.jpg", bytes.NewReader(tampered))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	rc, _, err := svc.Open(ctx, saved.File.ID, Caller{ProfileID: owner})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()
	stored, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if bytes.Contains(stored, []byte("Exif")) {
		t.Fatal("penanda EXIF masih ada setelah re-encode")
	}
}
