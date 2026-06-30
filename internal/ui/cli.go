package ui

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/madhank93/kubelings/internal/course"
)

// Shell opens an interactive shell wired to the cluster for a lesson (by name),
// showing the task and exposing task/hint/verify/solution commands. Used by the
// `kubelings shell <lesson>` CLI subcommand and mirrors the TUI's `t` action.
func Shell(root, lessonName string) error {
	mods, err := course.Discover(root)
	if err != nil {
		return err
	}
	var found *course.Lesson
	for _, m := range mods {
		for i := range m.Lessons {
			if m.Lessons[i].Name == lessonName {
				found = &m.Lessons[i]
			}
		}
	}
	if found == nil {
		return fmt.Errorf("no lesson named %q", lessonName)
	}
	kubeconfig, rc, err := shellEnv(root, found)
	if err != nil {
		return fmt.Errorf("prepare shell (is the cluster up?): %w", err)
	}
	c := exec.Command("bash", "--rcfile", rc, "-i")
	c.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}
