package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/madhank93/kubelings/internal/course"
)

// shellEnv prepares an interactive shell wired to the kind cluster for a lesson:
// an isolated KUBECONFIG (no global context mutation), the lesson task/hint/
// solution written to files, and helper commands. Returns (kubeconfig, rcfile).
func shellEnv(repoRoot string, l *course.Lesson) (string, string, error) {
	dir, err := os.MkdirTemp("", "kubelings-shell")
	if err != nil {
		return "", "", err
	}

	cluster := os.Getenv("KUBELINGS_CLUSTER")
	if cluster == "" {
		cluster = "kubelings"
	}
	out, err := exec.Command("kind", "get", "kubeconfig", "--name", cluster).Output()
	if err != nil {
		return "", "", err
	}
	kubeconfig := filepath.Join(dir, "kubeconfig")
	if err := os.WriteFile(kubeconfig, out, 0o600); err != nil {
		return "", "", err
	}

	title, name, task, hint, sol := "Kubelings shell", "", "", "(no hint)", "(no solution)"
	if l != nil {
		title, name = l.Title, l.Name
		if l.Task != "" {
			task = l.Task
		}
		if l.Hint != "" {
			hint = l.Hint
		}
		if l.Solution != "" {
			sol = l.Solution
		}
	}
	_ = os.WriteFile(filepath.Join(dir, "task.md"), []byte(task), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "hint.md"), []byte(hint), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "solution.md"), []byte(sol), 0o600)

	rc := filepath.Join(dir, "rc")
	if err := os.WriteFile(rc, []byte(buildRC(repoRoot, name, dir, title)), 0o600); err != nil {
		return "", "", err
	}
	return kubeconfig, rc, nil
}

// buildRC produces the interactive bash rcfile body. Separated for testing.
func buildRC(repoRoot, lesson, klDir, title string) string {
	return fmt.Sprintf(`
source ~/.bashrc 2>/dev/null || true
cd %[1]q
kubectl config set-context --current --namespace=kubelings >/dev/null 2>&1
alias k=kubectl
REPO=%[1]q; LESSON=%[2]q; KLDIR=%[3]q
task()     { cat "$KLDIR/task.md"; }
hint()     { cat "$KLDIR/hint.md"; }
solution() { cat "$KLDIR/solution.md"; }
verify()   { ( cd "$REPO" && scripts/run-challenge-local.sh "$LESSON" verify ); }
klreset()  { ( cd "$REPO" && scripts/run-challenge-local.sh "$LESSON" reset ); }
PS1='\[\e[36m\]kubelings\[\e[0m\]:%[2]s \w$ '
clear
printf '\e[1;36m%%s\e[0m\n\n' %[4]q
task
printf '\n\e[2mcommands:\e[0m \e[36mtask\e[0m · \e[36mhint\e[0m · \e[36mverify\e[0m · \e[36msolution\e[0m · \e[36mklreset\e[0m · k=kubectl · exit\n'
`, repoRoot, lesson, klDir, title)
}
