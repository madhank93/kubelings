package course

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot walks up to the dir containing courses/kubelings.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := filepath.Abs(".")
	for {
		if fi, err := os.Stat(filepath.Join(dir, "courses", "kubelings")); err == nil && fi.IsDir() {
			return dir
		}
		p := filepath.Dir(dir)
		if p == dir {
			t.Fatal("repo root not found")
		}
		dir = p
	}
}

func TestDiscover(t *testing.T) {
	root := repoRoot(t)
	mods, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(mods) < 2 {
		t.Fatalf("want >=2 modules, got %d", len(mods))
	}
	var runnable int
	var sawHint, sawSolution bool
	for _, m := range mods {
		for _, l := range m.Lessons {
			if l.Title == "" || l.Name == "" {
				t.Errorf("lesson missing title/name in %s", l.Dir)
			}
			if l.HasTasks {
				runnable++
				if l.Hint != "" {
					sawHint = true
				}
				if l.Solution != "" {
					sawSolution = true
				}
			}
		}
	}
	if runnable < 7 {
		t.Errorf("want >=7 runnable lessons, got %d", runnable)
	}
	if !sawHint {
		t.Error("no lesson hint extracted")
	}
	if !sawSolution {
		t.Error("no lesson solution extracted")
	}
}

func TestExtractDetails(t *testing.T) {
	root := repoRoot(t)
	mods, _ := Discover(root)
	for _, m := range mods {
		for _, l := range m.Lessons {
			if l.Name == "rolling-update" {
				if !strings.Contains(strings.ToLower(l.Solution), "maxsurge") {
					t.Errorf("rolling-update solution missing expected content: %q", l.Solution)
				}
				return
			}
		}
	}
}
