package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The generated rcfile must be valid bash and expose the helper commands.
func TestBuildRC(t *testing.T) {
	body := buildRC("/repo/kubelings", "rolling-update", "/tmp/kl", "Fix the Rolling Update: x")
	for _, want := range []string{"task()", "hint()", "verify()", "solution()", "alias k=kubectl"} {
		if !strings.Contains(body, want) {
			t.Errorf("rc missing %q", want)
		}
	}
	dir := t.TempDir()
	rc := filepath.Join(dir, "rc")
	if err := os.WriteFile(rc, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("bash", "-n", rc).CombinedOutput(); err != nil {
		t.Fatalf("bash -n failed: %v\n%s", err, out)
	}
}
