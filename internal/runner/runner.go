// Package runner is a thin wrapper over scripts/run-challenge-local.sh — the
// single execution engine. The TUI never reimplements lesson logic.
package runner

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const script = "scripts/run-challenge-local.sh"

// safeArg rejects a runner argument that the bash script would misread. Every
// argument it takes is a bare word — a lesson name or a verb — so anything
// carrying a path separator, or leading with a dash the script would parse as
// a flag, is a bug upstream and must not reach the shell.
func safeArg(a string) bool {
	return a != "" && !strings.ContainsAny(a, "/\\") && !strings.HasPrefix(a, "-")
}

// Cmd builds an *exec.Cmd for the runner with the given args, rooted at root.
// Unsafe arguments are dropped: the course loader already refuses lesson names
// of the wrong shape, and this is the structural backstop for any future call
// path that skips it.
func Cmd(root string, args ...string) *exec.Cmd {
	clean := make([]string, 0, len(args))
	for _, a := range args {
		if safeArg(a) {
			clean = append(clean, a)
		}
	}
	c := exec.Command("bash", append([]string{filepath.Join(root, script)}, clean...)...)
	c.Dir = root
	return c
}

// Capture runs the runner and returns combined output + whether it exited 0.
func Capture(root string, args ...string) (string, bool) {
	out, err := Cmd(root, args...).CombinedOutput()
	return string(out), err == nil
}

// CaptureContext is Capture with cancellation. Lesson inits run up to 360s, so
// the TUI needs a way out that isn't "quit the whole program".
//
// The runner shells out to kubectl/kind/docker, so killing the bash process
// alone would orphan those children. Run it in its own process group and signal
// the group, which is what actually stops the work.
func CaptureContext(ctx context.Context, root string, args ...string) (string, bool) {
	c := Cmd(root, args...)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var buf strings.Builder
	c.Stdout, c.Stderr = &buf, &buf
	if err := c.Start(); err != nil {
		return err.Error(), false
	}

	done := make(chan error, 1)
	go func() { done <- c.Wait() }()

	select {
	case err := <-done:
		return buf.String(), err == nil
	case <-ctx.Done():
		// Negative pid = the whole group. SIGKILL, not SIGTERM: the point of a
		// cancel is that it takes effect now.
		_ = syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		<-done
		return buf.String() + "\n\n^ cancelled — the cluster may be mid-change; press r to reset the scenario.", false
	}
}

// ClusterStatus reports whether the kind cluster is up, its node count, the
// Kubernetes server version, and the context name.
type ClusterStatus struct {
	Up      bool
	Nodes   int
	Version string
	Context string
}

func cluster() string {
	if v := os.Getenv("KUBELINGS_CLUSTER"); v != "" {
		return v
	}
	return "kubelings"
}

// Status queries kind/kubectl for the cluster state.
func Status() ClusterStatus {
	name := cluster()
	st := ClusterStatus{Context: "kind-" + name}
	out, err := exec.Command("kind", "get", "clusters").Output()
	if err != nil {
		return st
	}
	for _, l := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(l) == name {
			st.Up = true
		}
	}
	if !st.Up {
		return st
	}
	if nodes, err := exec.Command("kind", "get", "nodes", "--name", name).Output(); err == nil {
		for _, l := range strings.Split(strings.TrimSpace(string(nodes)), "\n") {
			if strings.TrimSpace(l) != "" {
				st.Nodes++
			}
		}
	}
	st.Version = serverVersion(st.Context)
	return st
}

// serverVersion returns the Kubernetes server gitVersion for the given context
// (explicit --context so it never queries the host's current-context).
func serverVersion(context string) string {
	out, err := exec.Command("kubectl", "--context", context, "version", "-o", "json").Output()
	if err != nil {
		return ""
	}
	var v struct {
		ServerVersion struct {
			GitVersion string `json:"gitVersion"`
		} `json:"serverVersion"`
	}
	if json.Unmarshal(out, &v) != nil {
		return ""
	}
	return v.ServerVersion.GitVersion
}
