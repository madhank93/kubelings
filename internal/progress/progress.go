// Package progress reads/writes per-lesson progress markers shared with the bash
// runner (.labctl/progress.tsv: <lesson>\t<state>\t<epoch>).
package progress

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type State string

const (
	None    State = "none"
	Started State = "started"
	Solved  State = "solved"
)

// Marker returns the glyph for a state.
func (s State) Marker() string {
	switch s {
	case Solved:
		return "✓"
	case Started:
		return "◐"
	default:
		return "◌"
	}
}

func file(root string) string { return filepath.Join(root, ".labctl", "progress.tsv") }

// Load reads the progress map (lesson -> state). Missing file = empty map.
func Load(root string) map[string]State {
	m := map[string]State{}
	b, err := os.ReadFile(file(root))
	if err != nil {
		return m
	}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Split(line, "\t")
		if len(f) >= 2 && f[0] != "" {
			m[f[0]] = State(f[1])
		}
	}
	return m
}

// Get returns the state for a lesson (None if unknown).
func Get(m map[string]State, lesson string) State {
	if s, ok := m[lesson]; ok {
		return s
	}
	return None
}

// Set writes a lesson's state (last-write-wins), matching the bash runner's
// format. None removes the row.
func Set(root, lesson string, s State) error {
	cur := Load(root)
	if s == None {
		delete(cur, lesson)
	} else {
		cur[lesson] = s
	}
	var b strings.Builder
	now := time.Now().Unix()
	for l, st := range cur {
		fmt.Fprintf(&b, "%s\t%s\t%d\n", l, st, now)
	}
	if err := os.MkdirAll(filepath.Dir(file(root)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file(root), []byte(b.String()), 0o644)
}

// StartedLessons returns lessons currently in the Started state.
func StartedLessons(m map[string]State) []string {
	var out []string
	for l, s := range m {
		if s == Started {
			out = append(out, l)
		}
	}
	return out
}
