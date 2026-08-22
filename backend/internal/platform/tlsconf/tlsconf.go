// Package tlsconf builds the server TLS configuration for production. The
// origin listens on 443 behind Cloudflare in Full (strict) mode: it presents a
// Cloudflare Origin Certificate and, per research R-01, verifies that the
// connecting client is Cloudflare itself through Authenticated Origin Pulls, so
// a request that bypasses the edge and hits the origin directly is rejected at
// the TLS layer.
package tlsconf

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// Load reads the origin certificate/key pair and the Cloudflare client CA, then
// returns a tls.Config that requires and verifies a client certificate signed
// by that CA. Any read or parse failure is returned so serve refuses to start
// rather than listening without the intended client-cert enforcement.
func Load(certPath, keyPath, clientCAPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("memuat sertifikat origin: %w", err)
	}

	caPEM, err := os.ReadFile(clientCAPath)
	if err != nil {
		return nil, fmt.Errorf("membaca CA klien Cloudflare: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("CA klien Cloudflare tidak memuat sertifikat yang sah")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
