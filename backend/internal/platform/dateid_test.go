package platform

import (
	"testing"
	"time"
)

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
