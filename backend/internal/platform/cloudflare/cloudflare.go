// Package cloudflare pins Cloudflare's published IP ranges as a compile-time
// constant and derives the real client IP from a request. The origin-address
// header (CF-Connecting-IP) is trusted only when the connection itself comes
// from one of these ranges; a connection from anywhere else is treated as
// direct and its headers are ignored. See docs/cloudflare-ips.md for the source
// URLs and retrieval date, and research.md R-01 for the rationale.
package cloudflare

import (
	"fmt"
	"net"
	"net/http"
)

// RetrievedAt is the date the ranges below were fetched from cloudflare.com.
// It must match the "Diambil" line in docs/cloudflare-ips.md; a test enforces
// this so the pinned list and the document cannot drift apart.
const RetrievedAt = "2026-08-22 12:14 UTC"

// ipv4CIDRs and ipv6CIDRs are transcribed verbatim from the official lists at
// https://www.cloudflare.com/ips-v4 and /ips-v6. They are parsed once in init;
// a malformed entry panics at startup so a typo fails loudly rather than
// silently widening or narrowing the trusted set.
var ipv4CIDRs = []string{
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"172.64.0.0/13",
	"131.0.72.0/22",
}

var ipv6CIDRs = []string{
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
}

// ranges holds the parsed CIDRs. Built once at package init.
var ranges []*net.IPNet

func init() {
	for _, cidr := range append(append([]string{}, ipv4CIDRs...), ipv6CIDRs...) {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("cloudflare: CIDR tidak sah %q: %v", cidr, err))
		}
		ranges = append(ranges, network)
	}
}

// inRange reports whether ip falls inside any pinned Cloudflare range.
func inRange(ip net.IP) bool {
	for _, network := range ranges {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// RealIP returns the client address to use for per-IP rate limiting and audit
// logs. It splits RemoteAddr, returns empty when that host cannot be parsed,
// returns the raw host untouched when the connection is not from a Cloudflare
// range (a direct connection: no header is trusted), and only then trusts
// CF-Connecting-IP. Follows research.md:70-84.
func RealIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}
	if !inRange(ip) {
		return host
	}
	if cf := net.ParseIP(r.Header.Get("CF-Connecting-IP")); cf != nil {
		return cf.String()
	}
	return host
}
