// Package runner is a thin wrapper over scripts/run-challenge-local.sh — the
// single execution engine. The TUI never reimplements lesson logic.
package runner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const script = "scripts/run-challenge-local.sh"

// Cmd builds an *exec.Cmd for the runner with the given args, rooted at root.
func Cmd(root string, args ...string) *exec.Cmd {
	c := exec.Command("bash", append([]string{filepath.Join(root, script)}, args...)...)
	c.Dir = root
	return c
}

// Capture runs the runner and returns combined output + whether it exited 0.
func Capture(root string, args ...string) (string, bool) {
	out, err := Cmd(root, args...).CombinedOutput()
	return string(out), err == nil
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
