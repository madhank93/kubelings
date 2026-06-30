// Command kubelings is the interactive local TUI for running kubelings lessons on
// a kind cluster. UI only — it delegates execution to scripts/run-challenge-local.sh.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madhank93/kubelings/internal/course"
	"github.com/madhank93/kubelings/internal/preflight"
	"github.com/madhank93/kubelings/internal/progress"
	"github.com/madhank93/kubelings/internal/runner"
	"github.com/madhank93/kubelings/internal/ui"
)

func main() {
	root, err := findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kubelings:", err)
		os.Exit(1)
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list", "doctor", "--list", "--doctor":
			doctor(root)
			return
		case "shell":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: kubelings shell <lesson>")
				os.Exit(2)
			}
			if err := ui.Shell(root, os.Args[2]); err != nil {
				fmt.Fprintln(os.Stderr, "kubelings:", err)
				os.Exit(1)
			}
			return
		case "-h", "--help", "help":
			fmt.Println("kubelings              launch the TUI")
			fmt.Println("kubelings doctor       env, cluster status, lessons (no TUI)")
			fmt.Println("kubelings shell <l>    shell wired to the cluster for lesson <l>")
			return
		}
	}
	p := tea.NewProgram(ui.New(root), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "kubelings:", err)
		os.Exit(1)
	}
}

// findRoot walks up from the cwd (and the executable dir) to the repo root — the
// directory containing courses/kubelings — so the TUI runs from anywhere.
func findRoot() (string, error) {
	starts := []string{}
	if wd, err := os.Getwd(); err == nil {
		starts = append(starts, wd)
	}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	for _, start := range starts {
		dir := start
		for {
			if fi, err := os.Stat(filepath.Join(dir, "courses", "kubelings")); err == nil && fi.IsDir() {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "", fmt.Errorf("could not locate the kubelings repo (no courses/kubelings found above %s)", mustWd())
}

func mustWd() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// doctor prints environment, cluster status, and discovered lessons (no TUI).
func doctor(root string) {
	fmt.Println("repo:", root)
	st := runner.Status()
	if st.Up {
		fmt.Printf("cluster: up · %d nodes · %s\n", st.Nodes, st.Context)
	} else {
		fmt.Println("cluster: down (run: scripts/run-challenge-local.sh up)")
	}
	if issues := preflight.Check(); len(issues) > 0 {
		fmt.Println("issues:")
		for _, is := range issues {
			fmt.Printf("  ⚠ %s → %s\n", is.Msg, is.Fix)
		}
	} else {
		fmt.Println("prereqs: ok (kind, kubectl, yq, docker)")
	}
	mods, err := course.Discover(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "discover:", err)
		os.Exit(1)
	}
	prog := progress.Load(root)
	for _, m := range mods {
		title := m.Title
		if title == "" {
			title = m.Name
		}
		fmt.Printf("\n%s\n", title)
		for _, l := range m.Lessons {
			mk := "  "
			if l.HasTasks {
				mk = progress.Get(prog, l.Name).Marker() + " "
			}
			fmt.Printf("  %s%-16s %s\n", mk, l.Name, l.Title)
		}
	}
}
