// Package config loads and validates process configuration from the
// environment. It takes a getenv function rather than reading os.Getenv
// directly so tests do not mutate process state.
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// Env is the deployment environment. Unknown values are an error, never a
// silent fall-through to development.
type Env string

const (
	EnvDevelopment Env = "development"
	EnvProduction  Env = "production"
	EnvTest        Env = "test"
)

// Config holds validated settings. Secrets live here as strings; nothing in
// this package ever logs a value.
type Config struct {
	AppEnv     Env
	AppBaseURL string

	DatabaseURL string
	UploadPath  string

	UploadTotalLimitMB int
	UploadFileLimitMB  int

	// Production-only. Empty outside production.
	TLSCertPath    string
	TLSKeyPath     string
	CFClientCAPath string
	MailjetAPIKey  string
	MailjetSecret  string
	MailFrom       string
	WhatsAppNumber string
	SentryDSN      string
}

// IsProduction reports whether the app runs in production. seed:test-data and
// reset:test-data refuse to run when this is true.
func (c Config) IsProduction() bool { return c.AppEnv == EnvProduction }

// IsDevelopment reports whether the app runs in development. The /docs Swagger
// UI routes are registered only when this is true, so they are absent (not
// merely rejected) in production and fall to the existing 404.
func (c Config) IsDevelopment() bool { return c.AppEnv == EnvDevelopment }

const (
	defaultUploadTotalLimitMB = 500
	defaultUploadFileLimitMB  = 5
)

// Load reads and validates configuration. All missing required variables are
// collected and reported together in a single error that names them and never
// includes any value.
func Load(getenv func(string) string) (Config, error) {
	get := func(k string) string { return strings.TrimSpace(getenv(k)) }

	rawEnv := get("APP_ENV")
	if rawEnv == "" {
		return Config{}, fmt.Errorf("variabel lingkungan wajib belum diisi: APP_ENV")
	}
	env := Env(rawEnv)
	switch env {
	case EnvDevelopment, EnvProduction, EnvTest:
	default:
		return Config{}, fmt.Errorf("APP_ENV tidak dikenal: %q (pilih development, production, atau test)", rawEnv)
	}

	cfg := Config{
		AppEnv:         env,
		AppBaseURL:     get("APP_BASE_URL"),
		DatabaseURL:    get("DATABASE_URL"),
		UploadPath:     get("UPLOAD_PATH"),
		TLSCertPath:    get("TLS_CERT_PATH"),
		TLSKeyPath:     get("TLS_KEY_PATH"),
		CFClientCAPath: get("CF_CLIENT_CA_PATH"),
		MailjetAPIKey:  get("MAILJET_API_KEY"),
		MailjetSecret:  get("MAILJET_SECRET_KEY"),
		MailFrom:       get("MAIL_FROM"),
		WhatsAppNumber: get("WHATSAPP_NUMBER"),
		SentryDSN:      get("SENTRY_DSN"),
	}

	var missing []string
	require := func(name, val string) {
		if val == "" {
			missing = append(missing, name)
		}
	}

	// Required in every environment.
	require("APP_BASE_URL", cfg.AppBaseURL)
	require("DATABASE_URL", cfg.DatabaseURL)
	require("UPLOAD_PATH", cfg.UploadPath)

	// Required only in production.
	if cfg.IsProduction() {
		require("TLS_CERT_PATH", cfg.TLSCertPath)
		require("TLS_KEY_PATH", cfg.TLSKeyPath)
		require("CF_CLIENT_CA_PATH", cfg.CFClientCAPath)
		require("MAILJET_API_KEY", cfg.MailjetAPIKey)
		require("MAILJET_SECRET_KEY", cfg.MailjetSecret)
		require("MAIL_FROM", cfg.MailFrom)
		require("WHATSAPP_NUMBER", cfg.WhatsAppNumber)
		require("SENTRY_DSN", cfg.SentryDSN)
	}

	total, err := parseLimit(get("UPLOAD_MAX_TOTAL_MB"), defaultUploadTotalLimitMB)
	if err != nil {
		return Config{}, fmt.Errorf("UPLOAD_MAX_TOTAL_MB: %w", err)
	}
	file, err := parseLimit(get("UPLOAD_MAX_FILE_MB"), defaultUploadFileLimitMB)
	if err != nil {
		return Config{}, fmt.Errorf("UPLOAD_MAX_FILE_MB: %w", err)
	}
	cfg.UploadTotalLimitMB = total
	cfg.UploadFileLimitMB = file

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("variabel lingkungan wajib belum diisi: %s",
			strings.Join(missing, ", "))
	}
	return cfg, nil
}

func parseLimit(raw string, def int) (int, error) {
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("bukan bilangan bulat: %q", raw)
	}
	if n <= 0 {
		return 0, fmt.Errorf("harus lebih dari nol: %d", n)
	}
	return n, nil
}
