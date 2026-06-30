package ui

import (
	"os"
	"os/exec"
	"path/filepath"
)

// writeShellEnv prepares an isolated KUBECONFIG for the kind cluster and a bash
// rcfile that selects the kubelings namespace. Returns (kubeconfigPath, rcPath).
// Using a dedicated KUBECONFIG means the drop-to-shell never mutates the user's
// global current-context.
func writeShellEnv(context string) (string, string, error) {
	dir, err := os.MkdirTemp("", "kubelings-shell")
	if err != nil {
		return "", "", err
	}
	kubeconfig := filepath.Join(dir, "kubeconfig")

	// Export the kind cluster's kubeconfig into the temp file.
	cluster := os.Getenv("KUBELINGS_CLUSTER")
	if cluster == "" {
		cluster = "kubelings"
	}
	out, err := exec.Command("kind", "get", "kubeconfig", "--name", cluster).Output()
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(kubeconfig, out, 0o600); err != nil {
		return "", "", err
	}

	rc := filepath.Join(dir, "rc")
	rcBody := `
source ~/.bashrc 2>/dev/null || true
kubectl config set-context --current --namespace=kubelings >/dev/null 2>&1
alias k=kubectl
PS1='kubelings:\w$ '
echo "kubelings shell — context: ` + context + `  ns: kubelings   (type 'exit' to return)"
`
	if err := os.WriteFile(rc, []byte(rcBody), 0o600); err != nil {
		return "", "", err
	}
	return kubeconfig, rc, nil
}
