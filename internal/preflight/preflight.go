// Package preflight checks that the local toolchain is ready and returns
// actionable issues for the UI to surface.
package preflight

import "os/exec"

// Issue is a missing prerequisite plus how to fix it.
type Issue struct {
	Msg string
	Fix string
}

// Check verifies required binaries exist and the Docker runtime is running.
func Check() []Issue {
	var issues []Issue
	bins := map[string]string{
		"kind":    "brew install kind",
		"kubectl": "brew install kubernetes-cli",
		"yq":      "brew install yq",
		"docker":  "install a Docker runtime (OrbStack)",
	}
	// Stable order for predictable banners.
	for _, b := range []string{"docker", "kind", "kubectl", "yq"} {
		if _, err := exec.LookPath(b); err != nil {
			issues = append(issues, Issue{Msg: b + " not found", Fix: bins[b]})
		}
	}
	// Docker daemon reachable?
	if _, err := exec.LookPath("docker"); err == nil {
		if err := exec.Command("docker", "info").Run(); err != nil {
			issues = append(issues, Issue{
				Msg: "Docker runtime not running",
				Fix: "start OrbStack (or Docker), then press g",
			})
		}
	}
	return issues
}
