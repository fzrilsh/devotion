package migrate

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// migrationsDir returns the absolute path to backend/db/migrations by walking
// up to the module root.
func migrationsDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "db", "migrations")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod tidak ditemukan di atas direktori kerja")
		}
		dir = parent
	}
}

// TestNoTimeDefaultInMigrations enforces the plan's rule: no DEFAULT now() or
// DEFAULT CURRENT_TIMESTAMP anywhere. Time defaults would let a row's timestamp
// come from the database wall clock instead of the injected Clock, which is the
// whole reason deadline logic is testable. DEFAULT false/true/0 stay allowed.
func TestNoTimeDefaultInMigrations(t *testing.T) {
	dir := migrationsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("baca direktori migrasi: %v", err)
	}
	bad := regexp.MustCompile(`(?i)default\s+(now\s*\(\s*\)|current_timestamp|localtimestamp|current_date|current_time|clock_timestamp\s*\(\s*\)|statement_timestamp\s*\(\s*\)|transaction_timestamp\s*\(\s*\))`)
	var offenders []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if bad.MatchString(string(src)) {
			offenders = append(offenders, e.Name())
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("DEFAULT waktu dilarang di migrasi, ditemukan di: %s",
			strings.Join(offenders, ", "))
	}
}

// TestMigrationPairsComplete verifies every up has a matching down and the set
// runs 1..15 without a gap.
func TestMigrationPairsComplete(t *testing.T) {
	dir := migrationsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	ups := map[string]bool{}
	downs := map[string]bool{}
	num := regexp.MustCompile(`^(\d{6})_.*\.(up|down)\.sql$`)
	for _, e := range entries {
		m := num.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		if m[2] == "up" {
			ups[m[1]] = true
		} else {
			downs[m[1]] = true
		}
	}
	if len(ups) != 15 {
		t.Fatalf("harap 15 migrasi up, ada %d", len(ups))
	}
	for i := 1; i <= 15; i++ {
		key := fmtNum(i)
		if !ups[key] {
			t.Errorf("migrasi up %s hilang", key)
		}
		if !downs[key] {
			t.Errorf("migrasi down %s hilang", key)
		}
	}
}

func fmtNum(i int) string {
	s := "000000"
	d := []byte(s)
	n := i
	for p := 5; p >= 0 && n > 0; p-- {
		d[p] = byte('0' + n%10)
		n /= 10
	}
	return string(d)
}
