package config

import (
	"strings"
	"testing"
)

// mapEnv returns a getenv backed by a map, so tests never touch process state.
func mapEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func baseDev() map[string]string {
	return map[string]string{
		"APP_ENV":      "development",
		"APP_BASE_URL": "http://localhost:8080",
		"DATABASE_URL": "postgres://localhost/devotion",
		"UPLOAD_PATH":  "/tmp/unggahan",
	}
}

func baseProd() map[string]string {
	return map[string]string{
		"APP_ENV":            "production",
		"APP_BASE_URL":       "https://devotion.cloud",
		"DATABASE_URL":       "postgres://db/devotion",
		"UPLOAD_PATH":        "/opt/devotion/unggahan",
		"TLS_CERT_PATH":      "/tls/origin.pem",
		"TLS_KEY_PATH":       "/tls/origin.key",
		"CF_CLIENT_CA_PATH":  "/tls/cf.pem",
		"MAILJET_API_KEY":    "k",
		"MAILJET_SECRET_KEY": "s",
		"MAIL_FROM":          "noreply@devotion.cloud",
		"WHATSAPP_NUMBER":    "62800",
		"SENTRY_DSN":         "https://sentry",
	}
}

func TestLoad_DevelopmentDefaults(t *testing.T) {
	cfg, err := Load(mapEnv(baseDev()))
	if err != nil {
		t.Fatalf("mau nil, dapat %v", err)
	}
	if cfg.UploadTotalLimitMB != 500 || cfg.UploadFileLimitMB != 5 {
		t.Fatalf("default limit salah: %d, %d", cfg.UploadTotalLimitMB, cfg.UploadFileLimitMB)
	}
	if cfg.IsProduction() {
		t.Fatal("development tidak boleh IsProduction")
	}
}

func TestLoad_UnknownEnvIsError(t *testing.T) {
	m := baseDev()
	m["APP_ENV"] = "staging"
	if _, err := Load(mapEnv(m)); err == nil {
		t.Fatal("APP_ENV tak dikenal harus galat, bukan jatuh ke development")
	}
}

func TestLoad_CollectsAllMissingProductionVars(t *testing.T) {
	m := baseProd()
	delete(m, "TLS_CERT_PATH")
	delete(m, "MAILJET_API_KEY")
	delete(m, "SENTRY_DSN")
	_, err := Load(mapEnv(m))
	if err == nil {
		t.Fatal("mau galat karena tiga variabel hilang")
	}
	for _, name := range []string{"TLS_CERT_PATH", "MAILJET_API_KEY", "SENTRY_DSN"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("galat tidak menyebut %s: %v", name, err)
		}
	}
}

func TestLoad_ProductionValid(t *testing.T) {
	cfg, err := Load(mapEnv(baseProd()))
	if err != nil {
		t.Fatalf("mau nil, dapat %v", err)
	}
	if !cfg.IsProduction() {
		t.Fatal("mau IsProduction true")
	}
}

func TestLoad_DevDoesNotRequireProductionVars(t *testing.T) {
	// Dev has no TLS/Mailjet/etc and must still load.
	if _, err := Load(mapEnv(baseDev())); err != nil {
		t.Fatalf("dev tidak boleh menuntut variabel produksi: %v", err)
	}
}

func TestLoad_ErrorNeverLeaksValues(t *testing.T) {
	m := baseDev()
	m["DATABASE_URL"] = "" // secret-ish; must be named but not its (empty) value
	delete(m, "APP_BASE_URL")
	_, err := Load(mapEnv(m))
	if err == nil {
		t.Fatal("mau galat")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") || !strings.Contains(err.Error(), "APP_BASE_URL") {
		t.Fatalf("galat harus menyebut nama yang hilang: %v", err)
	}
}

func TestLoad_BadLimitIsError(t *testing.T) {
	m := baseDev()
	m["UPLOAD_MAX_FILE_MB"] = "-3"
	if _, err := Load(mapEnv(m)); err == nil {
		t.Fatal("limit negatif harus galat")
	}
}

func TestLoad_ReadsUploadLimitEnvNames(t *testing.T) {
	m := baseDev()
	m["UPLOAD_MAX_TOTAL_MB"] = "300"
	m["UPLOAD_MAX_FILE_MB"] = "7"
	cfg, err := Load(mapEnv(m))
	if err != nil {
		t.Fatalf("mau nil, dapat %v", err)
	}
	if cfg.UploadTotalLimitMB != 300 {
		t.Errorf("UPLOAD_MAX_TOTAL_MB tidak terbaca: mau 300, dapat %d", cfg.UploadTotalLimitMB)
	}
	if cfg.UploadFileLimitMB != 7 {
		t.Errorf("UPLOAD_MAX_FILE_MB tidak terbaca: mau 7, dapat %d", cfg.UploadFileLimitMB)
	}
}
