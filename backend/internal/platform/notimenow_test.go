package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoTimeNowOutsidePlatformAndCmd enforces CLAUDE.md rule 5: business logic
// must take time from an injected Clock, never time.Now(). Only platform (which
// defines SystemClock) and cmd (process entry) may call it directly. This sweep
// is the only mechanism that catches a violation in the branch that introduces
// it; review alone misses it.
func TestNoTimeNowOutsidePlatformAndCmd(t *testing.T) {
	root := repoRoot(t)
	var offenders []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == "platform" || base == "cmd" || base == "webdist" || base == "sqlcgen" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(src), "time.Now()") {
			rel, _ := filepath.Rel(root, path)
			offenders = append(offenders, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("menyisir tree: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("time.Now() dilarang di luar platform dan cmd, ditemukan di: %s",
			strings.Join(offenders, ", "))
	}
}

// repoRoot returns the backend/ module root by walking up to the go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod tidak ditemukan di atas direktori kerja")
		}
		dir = parent
	}
}
