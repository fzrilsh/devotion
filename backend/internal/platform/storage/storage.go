// Package storage stores and serves the two kinds of uploaded file the platform
// accepts: identity documents and business location photos. It is the only
// place bytes cross from an untrusted client onto disk, so every check that
// keeps a hostile or oversized upload from hurting the 2GB box lives here.
//
// The order of checks in Save is deliberate and load-bearing. The reader is
// bounded to the per-file limit before a single byte is decoded, so a decode
// bomb cannot exhaust memory. The content type is read from the leading magic
// bytes, never from the client's filename or Content-Type, so a .jpg that is
// really something else is refused. Images are decoded and re-encoded, which is
// the reliable way to drop EXIF (a location photo taken on a phone carries GPS
// coordinates, and for a home-run konveksi that is someone's home address). The
// owner's running total is checked against the 500MB quota before the write.
// Only then is the file written under a system-generated UUID name.
//
// Files are never reachable by a static path. Open resolves a file by id and
// enforces owner-or-admin (FR-009) before handing back a reader, so the access
// check cannot be skipped by guessing a URL.
package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// Sentinel errors let handlers map a failure to its problem code without
// inspecting strings. They mirror the three obligations T017 must prove: an
// oversized file, a file whose real type is not allowed, and a full quota.
// ErrForbidden and ErrNotFound back the owner-or-admin gate in Open.
var (
	// ErrTooLarge means the upload exceeded the per-file limit. The reader is
	// bounded, so this is detected without buffering the whole oversize body.
	ErrTooLarge = errors.New("storage: berkas melebihi batas ukuran")
	// ErrUnsupportedType means the leading bytes did not match an allowed type.
	// The client's filename and Content-Type are never consulted.
	ErrUnsupportedType = errors.New("storage: tipe berkas tidak diizinkan")
	// ErrQuotaFull means storing this file would push the owner past the total
	// quota. The message a handler shows must be clear (T017), so this is a
	// distinct error, not a generic write failure.
	ErrQuotaFull = errors.New("storage: kuota penyimpanan penuh")
	// ErrForbidden means the caller is neither the file's owner nor an admin.
	ErrForbidden = errors.New("storage: bukan pemilik dan bukan admin")
	// ErrNotFound means no file has the given id.
	ErrNotFound = errors.New("storage: berkas tidak ditemukan")
)

// allowedTypes are the MIME types the uploaded_file.allowed_type CHECK permits.
// Only the two image types are re-encoded; a PDF is validated by its magic
// bytes and written verbatim, since it carries no EXIF to strip.
const (
	mimeJPEG = "image/jpeg"
	mimePNG  = "image/png"
	mimePDF  = "application/pdf"
)

// Service holds the on-disk root and the limits, plus the pool for the file
// rows. It carries no per-request state, so one instance is shared.
type Service struct {
	pool       *pgxpool.Pool
	clock      platform.Clock
	root       string
	fileLimit  int64
	totalLimit int64
}

// New builds a Service. root is the upload directory (config UploadPath);
// fileLimitMB and totalLimitMB are the per-file and per-owner ceilings in
// megabytes (config defaults 5 and 500). The directory is created if absent so
// the first upload does not fail on a missing path.
func New(pool *pgxpool.Pool, clock platform.Clock, root string, fileLimitMB, totalLimitMB int) (*Service, error) {
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("storage: menyiapkan direktori unggahan: %w", err)
	}
	return &Service{
		pool:       pool,
		clock:      clock,
		root:       root,
		fileLimit:  int64(fileLimitMB) * 1024 * 1024,
		totalLimit: int64(totalLimitMB) * 1024 * 1024,
	}, nil
}

func (s *Service) queries() *sqlcgen.Queries { return sqlcgen.New(s.pool) }

// Caller is the authenticated principal Open checks against a file's owner. A
// caller with no profile (ProfileID invalid) can still be an admin; a
// non-admin with no profile matches no file and is refused.
type Caller struct {
	ProfileID pgtype.UUID
	IsAdmin   bool
}

// Saved is what Save returns: the stored row plus nothing the caller did not
// already have. Handlers build their response from this.
type Saved struct {
	File sqlcgen.UploadedFile
}

// Save validates and stores one uploaded file for ownerProfileID. src is the
// raw request body reader; originalName is the client's filename, kept only as
// display metadata. fileType is the caller's declared purpose (identity vs
// location), which is orthogonal to the content check: a location photo must
// still be a real image.
//
// The steps run in the order the package comment fixes. Any check that fails
// returns before a byte is written, so a rejected upload leaves no file behind.
func (s *Service) Save(ctx context.Context, ownerProfileID pgtype.UUID, fileType sqlcgen.FileType, originalName string, src io.Reader) (Saved, error) {
	// Bound the reader to one byte past the limit before decoding anything, so
	// an oversized or malicious body cannot exhaust memory. Reading limit+1
	// bytes lets us tell "exactly at the limit" from "over".
	limited := io.LimitReader(src, s.fileLimit+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return Saved{}, fmt.Errorf("storage: membaca unggahan: %w", err)
	}
	if int64(len(raw)) > s.fileLimit {
		return Saved{}, ErrTooLarge
	}
	if len(raw) == 0 {
		return Saved{}, ErrUnsupportedType
	}

	// Sniff the real type from the leading bytes. http.DetectContentType reads
	// at most 512 bytes and never trusts the filename or a client header.
	sniffed := http.DetectContentType(raw)
	mime, canonical, err := s.normalize(sniffed, raw)
	if err != nil {
		return Saved{}, err
	}

	// Check the owner's running total against the quota before writing. A
	// concurrent upload could still race past the ceiling, but the per-file
	// limit bounds the overshoot and the check keeps ordinary use honest.
	used, err := s.queries().SumUploadedBytesByOwner(ctx, ownerProfileID)
	if err != nil {
		return Saved{}, fmt.Errorf("storage: menghitung kuota: %w", err)
	}
	if used+int64(len(canonical)) > s.totalLimit {
		return Saved{}, ErrQuotaFull
	}

	// Write under a system-generated UUID-style name, never the client's. The
	// extension is derived from the verified type, not the client's filename.
	name := randomName() + extForMIME(mime)
	dest := filepath.Join(s.root, name)
	if err := os.WriteFile(dest, canonical, 0o640); err != nil {
		return Saved{}, fmt.Errorf("storage: menulis berkas: %w", err)
	}

	row, err := s.queries().CreateUploadedFile(ctx, sqlcgen.CreateUploadedFileParams{
		OwnerProfileID: ownerProfileID,
		Type:           fileType,
		OriginalName:   originalName,
		MimeType:       mime,
		SizeBytes:      int32(len(canonical)),
		StoragePath:    name,
		CreatedAt:      pgtype.Timestamptz{Time: s.clock.Now(), Valid: true},
	})
	if err != nil {
		// Roll back the on-disk file so a failed insert does not orphan bytes.
		_ = os.Remove(dest)
		return Saved{}, fmt.Errorf("storage: menyimpan baris berkas: %w", err)
	}
	return Saved{File: row}, nil
}

// normalize verifies the sniffed type is allowed and returns the canonical MIME
// plus the bytes to store. Images are decoded and re-encoded to drop EXIF and
// any other trailing metadata; a PDF is validated by its sniffed type and
// stored verbatim. The sniffed type, not the client's, decides everything.
func (s *Service) normalize(sniffed string, raw []byte) (mime string, canonical []byte, err error) {
	switch {
	case sniffed == mimeJPEG:
		img, derr := jpeg.Decode(bytes.NewReader(raw))
		if derr != nil {
			return "", nil, ErrUnsupportedType
		}
		var buf bytes.Buffer
		if eerr := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); eerr != nil {
			return "", nil, fmt.Errorf("storage: re-encode jpeg: %w", eerr)
		}
		return mimeJPEG, buf.Bytes(), nil
	case sniffed == mimePNG:
		img, derr := png.Decode(bytes.NewReader(raw))
		if derr != nil {
			return "", nil, ErrUnsupportedType
		}
		var buf bytes.Buffer
		if eerr := (&png.Encoder{CompressionLevel: png.DefaultCompression}).Encode(&buf, img); eerr != nil {
			return "", nil, fmt.Errorf("storage: re-encode png: %w", eerr)
		}
		return mimePNG, buf.Bytes(), nil
	case sniffed == mimePDF || sniffed == "application/pdf":
		return mimePDF, raw, nil
	default:
		return "", nil, ErrUnsupportedType
	}
}

// Open resolves a file by id and enforces owner-or-admin before returning a
// reader over its bytes. This is the only path to a stored file; there is no
// static route, so the access check (FR-009) cannot be bypassed. A non-owner
// non-admin gets ErrForbidden, never a reader, and a missing id is ErrNotFound.
func (s *Service) Open(ctx context.Context, id pgtype.UUID, caller Caller) (io.ReadCloser, sqlcgen.UploadedFile, error) {
	row, err := s.queries().GetUploadedFile(ctx, id)
	if err != nil {
		return nil, sqlcgen.UploadedFile{}, ErrNotFound
	}
	if !caller.IsAdmin {
		if !caller.ProfileID.Valid || row.OwnerProfileID.Bytes != caller.ProfileID.Bytes {
			return nil, sqlcgen.UploadedFile{}, ErrForbidden
		}
	}
	f, err := os.Open(filepath.Join(s.root, row.StoragePath))
	if err != nil {
		return nil, sqlcgen.UploadedFile{}, ErrNotFound
	}
	return f, row, nil
}

// randomName returns a 32-hex-character name from crypto/rand. It is not a UUID
// but serves the same purpose: an unguessable, collision-free filename that
// carries none of the client's chosen name.
func randomName() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// extForMIME maps a verified MIME to its file extension. The extension is
// cosmetic since files are served through a handler, but a correct one keeps
// the on-disk directory legible.
func extForMIME(mime string) string {
	switch mime {
	case mimeJPEG:
		return ".jpg"
	case mimePNG:
		return ".png"
	case mimePDF:
		return ".pdf"
	default:
		return ""
	}
}
