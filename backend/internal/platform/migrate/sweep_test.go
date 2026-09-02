package migrate

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
// runs 1..N without a gap, where N is the highest version present in the
// directory. It reads the count from the directory instead of hardcoding it, so
// adding a migration does not turn this red on its own; it stays red only if a
// pair is missing or a number is skipped.
func TestMigrationPairsComplete(t *testing.T) {
	dir := migrationsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	ups := map[int]bool{}
	downs := map[int]bool{}
	highest := 0
	num := regexp.MustCompile(`^(\d{6})_.*\.(up|down)\.sql$`)
	for _, e := range entries {
		m := num.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("nomor migrasi tidak sah pada %s: %v", e.Name(), err)
		}
		if m[2] == "up" {
			ups[n] = true
		} else {
			downs[n] = true
		}
		if n > highest {
			highest = n
		}
	}
	if highest == 0 {
		t.Fatal("tidak ada berkas migrasi ditemukan")
	}
	for i := 1; i <= highest; i++ {
		if !ups[i] {
			t.Errorf("migrasi up %s hilang", fmtNum(i))
		}
		if !downs[i] {
			t.Errorf("migrasi down %s hilang", fmtNum(i))
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
