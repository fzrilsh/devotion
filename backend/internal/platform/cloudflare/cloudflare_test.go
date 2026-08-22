package cloudflare

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestRetrievedAt_MatchesDoc keeps the pinned constant and docs/cloudflare-ips.md
// in sync: if someone refreshes the list they must update both.
func TestRetrievedAt_MatchesDoc(t *testing.T) {
	data, err := os.ReadFile("../../../../docs/cloudflare-ips.md")
	if err != nil {
		t.Fatalf("baca docs/cloudflare-ips.md: %v", err)
	}
	re := regexp.MustCompile(`(?m)^Diambil:\s*(.+?)\s*$`)
	m := re.FindStringSubmatch(string(data))
	if m == nil {
		t.Fatal("baris 'Diambil:' tidak ditemukan di docs/cloudflare-ips.md")
	}
	if got := strings.TrimSpace(m[1]); got != RetrievedAt {
		t.Fatalf("tanggal dokumen %q != RetrievedAt %q", got, RetrievedAt)
	}
}

// TestRealIP_TrustsHeaderOnlyFromCloudflareRange proves the trust boundary: a
// connection from a Cloudflare range gets its CF-Connecting-IP honored, while a
// direct connection from outside the range never has its header trusted.
func TestRealIP_TrustsHeaderOnlyFromCloudflareRange(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		cfHeader   string
		want       string
	}{
		{"dari cloudflare, header dipercaya", "173.245.48.1:443", "203.0.113.9", "203.0.113.9"},
		{"koneksi langsung, header diabaikan", "203.0.113.9:1234", "8.8.8.8", "203.0.113.9"},
		{"dari cloudflare tanpa header", "162.158.0.5:443", "", "162.158.0.5"},
		{"dari cloudflare, header rusak", "104.16.0.1:443", "bukan-ip", "104.16.0.1"},
		{"remoteaddr tak terurai", "tidak-valid", "1.1.1.1", ""},
		{"ipv6 cloudflare, header dipercaya", "[2400:cb00::1]:443", "198.51.100.7", "198.51.100.7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{RemoteAddr: tc.remoteAddr, Header: http.Header{}}
			if tc.cfHeader != "" {
				r.Header.Set("CF-Connecting-IP", tc.cfHeader)
			}
			if got := RealIP(r); got != tc.want {
				t.Fatalf("RealIP() = %q, mau %q", got, tc.want)
			}
		})
	}
}

// TestRanges_AllParsed confirms init parsed every pinned CIDR.
func TestRanges_AllParsed(t *testing.T) {
	if want := len(ipv4CIDRs) + len(ipv6CIDRs); len(ranges) != want {
		t.Fatalf("ranges punya %d entri, mau %d", len(ranges), want)
	}
}
