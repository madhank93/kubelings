// Package progress reads/writes per-lesson progress markers shared with the bash
// runner (.labctl/progress.tsv: <lesson>\t<state>\t<epoch>).
package progress

import (
	"os"
	"path/filepath"
	"strings"
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
