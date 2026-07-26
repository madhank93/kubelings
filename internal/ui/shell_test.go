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

func TestShellQuote(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", `'plain'`},
		{"two words", `'two words'`},
		{"$(id)", `'$(id)'`},
		{"it's", `'it'\''s'`},
		{"", `''`},
	}
	for _, c := range cases {
		if got := shellQuote(c.in); got != c.want {
			t.Errorf("shellQuote(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

// Lesson titles and names come from course frontmatter and directory names.
// Interpolating them with Go's %q left them inside a double-quoted shell
// string, where $(…) still runs. The rcfile must treat them as literal text.
func TestBuildRCDoesNotExecuteInterpolatedValues(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "pwned")
	payload := `x$(touch ` + marker + `)y`

	body := buildRC("/repo/kubelings", "lesson", "/tmp/kl", payload)
	rc := filepath.Join(t.TempDir(), "rc")
	if err := os.WriteFile(rc, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	// Source the rcfile in a non-interactive shell. Only the substitution
	// matters here; the interactive-only bits are allowed to fail.
	cmd := exec.Command("bash", "-c", "source "+rc+" >/dev/null 2>&1; true")
	_ = cmd.Run()

	if _, err := os.Stat(marker); err == nil {
		t.Fatal("command substitution in a lesson title was executed by the rcfile")
	}
	if !strings.Contains(body, `'`+payload[:1]) {
		t.Errorf("payload not present as literal text in the rcfile")
	}
}
