package runner

import (
	"slices"
	"testing"
)

func TestSafeArg(t *testing.T) {
	ok := []string{"up", "down", "list", "rolling-update", "incident-node-oom", "verify"}
	for _, a := range ok {
		if !safeArg(a) {
			t.Errorf("safeArg(%q) = false, want true", a)
		}
	}
	bad := []string{"", "../../etc", "a/b", `a\b`, "--exec", "-rf"}
	for _, a := range bad {
		if safeArg(a) {
			t.Errorf("safeArg(%q) = true, want false", a)
		}
	}
}

// A lesson name that escaped the course loader must not reach the shell.
func TestCmdDropsUnsafeArgs(t *testing.T) {
	c := Cmd("/repo", "../../etc/passwd", "verify")
	if slices.Contains(c.Args, "../../etc/passwd") {
		t.Fatalf("unsafe argument survived into %v", c.Args)
	}
	if !slices.Contains(c.Args, "verify") {
		t.Errorf("safe argument was dropped from %v", c.Args)
	}
}
