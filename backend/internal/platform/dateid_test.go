package platform

import (
	"testing"
	"time"
)

// TestFormatDate_BentukISO proves the wire serializer renders a bare
// YYYY-MM-DD, that it is the exact inverse of ParseDate, and that it localizes
// to Asia/Jakarta so a date column decoded as UTC midnight still names its own
// day while a timestamptz names the WIB day. A JSON field the contract declares
// as `format: date` reads through this, never through FormatDateID.
func TestFormatDate_BentukISO(t *testing.T) {
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			name: "tanggal biasa",
			in:   time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC),
			want: "2026-08-24",
		},
		{
			name: "hari dan bulan satu digit tetap ber-nol depan",
			in:   time.Date(2026, 1, 5, 3, 0, 0, 0, time.UTC),
			want: "2026-01-05",
		},
		{
			name: "kolom date terbaca tengah malam UTC tetap hari itu",
			in:   time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), // 07:00 WIB, hari sama
			want: "2026-08-20",
		},
		{
			name: "menjelang tengah malam UTC bergeser ke hari WIB",
			in:   time.Date(2026, 12, 31, 20, 0, 0, 0, time.UTC), // 03:00 WIB, 1 Januari
			want: "2027-01-01",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatDate(tc.in); got != tc.want {
				t.Errorf("FormatDate(%v) = %q, mau %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestFormatDate_BolakBalikDenganParseDate proves the pair round-trips: a date a
// client sends comes back byte for byte. A field that takes ParseDate on the way
// in must give FormatDate back on the way out, or the same field is two
// different formats depending on direction.
func TestFormatDate_BolakBalikDenganParseDate(t *testing.T) {
	for _, s := range []string{"2026-08-20", "2026-01-05", "2027-01-01", "2026-12-31"} {
		t.Run(s, func(t *testing.T) {
			parsed, err := ParseDate(s)
			if err != nil {
				t.Fatalf("ParseDate(%q): %v", s, err)
			}
			if got := FormatDate(parsed); got != s {
				t.Errorf("FormatDate(ParseDate(%q)) = %q, mau %q", s, got, s)
			}
		})
	}
}

// TestFormatDateID_NamaBulanIndonesia proves the long date renders Indonesian
// month names, no zero-padded day, and localizes to Asia/Jakarta so a timestamp
// taken just before midnight UTC names the WIB day a reader would see.
func TestFormatDateID_NamaBulanIndonesia(t *testing.T) {
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			name: "tanggal biasa",
			in:   time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC),
			want: "24 Agustus 2026",
		},
		{
			name: "hari satu digit tanpa nol depan",
			in:   time.Date(2026, 1, 5, 3, 0, 0, 0, time.UTC),
			want: "5 Januari 2026",
		},
		{
			name: "menjelang tengah malam UTC bergeser ke hari WIB",
			in:   time.Date(2026, 12, 31, 20, 0, 0, 0, time.UTC), // 03:00 WIB, 1 Januari
			want: "1 Januari 2027",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatDateID(tc.in); got != tc.want {
				t.Errorf("FormatDateID(%v) = %q, mau %q", tc.in, got, tc.want)
			}
		})
	}
}
