package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// plain strips ANSI styling so assertions read the text, not the escape codes.
func plain(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func renderModel() model {
	m := modelWithCourse(twoModules())
	m.ready = true
	m.spin = spinner.New()
	return m
}

// The footer is the only place the filter is visible; without it the learner is
// typing into a list with no indication of why rows vanished.
func TestFooterShowsFilterQueryAndMatchCount(t *testing.T) {
	m := renderModel()
	m.filtering = true
	m.filter = "crash"
	m.rebuildRows()

	f := plain(m.footer())
	if !strings.Contains(f, "crash") {
		t.Errorf("footer does not echo the query:\n%s", f)
	}
	if !strings.Contains(f, "1 match") {
		t.Errorf("footer does not report the match count:\n%s", f)
	}
}

// After ↵ the filter stays applied but the keyboard returns to commands — the
// learner needs a standing reminder that the list is still narrowed.
func TestFooterShowsStandingFilterAfterCommit(t *testing.T) {
	m := renderModel()
	m.filter = "crash"
	m.filtering = false
	m.rebuildRows()

	f := plain(m.footer())
	if !strings.Contains(f, "1 of 3 lessons") {
		t.Errorf("committed filter not surfaced as N of M:\n%s", f)
	}
	if !strings.Contains(f, "esc clears") {
		t.Errorf("footer does not say how to clear the filter:\n%s", f)
	}
}

func TestFooterAdvertisesCancelWhileRunning(t *testing.T) {
	m := renderModel()
	m.running = true
	m.runLbl = "init crashloop-triage"

	f := plain(m.footer())
	if !strings.Contains(f, "esc") || !strings.Contains(f, "cancel") {
		t.Errorf("running footer does not offer a way out:\n%s", f)
	}
}

func TestFooterConfirmsClusterDeletion(t *testing.T) {
	m := renderModel()
	m.confirmDown = true

	f := plain(m.footer())
	if !strings.Contains(f, "delete the kind cluster?") {
		t.Errorf("destroy-cluster prompt missing:\n%s", f)
	}
	if !strings.Contains(f, "lost") {
		t.Errorf("prompt does not state the consequence:\n%s", f)
	}
}

// 'd' used to delete the cluster on the keystroke. It must now only arm a prompt.
func TestDownKeyOnlyArmsConfirmation(t *testing.T) {
	m := renderModel()
	got, _ := m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if !got.(model).confirmDown {
		t.Fatal("'d' did not arm the confirmation")
	}

	// 'n' (anything but y) backs out without deleting.
	m2 := got.(model)
	after, cmd := m2.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if after.(model).confirmDown {
		t.Error("'n' left the confirmation armed")
	}
	if cmd != nil {
		t.Error("'n' issued a command; declining must do nothing")
	}
}

// Typing a lesson name must not fire commands that happen to share those letters.
func TestFilterModeSwallowsCommandKeys(t *testing.T) {
	m := renderModel()
	m.filtering = true

	for _, r := range "daemonset" {
		next, _ := m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(model)
	}
	if m.filter != "daemonset" {
		t.Errorf("filter = %q, want %q", m.filter, "daemonset")
	}
	if m.confirmDown {
		t.Error("typing 'd' inside the filter armed cluster deletion")
	}
	if !m.filtering {
		t.Error("typing 'q' inside the filter left filter mode")
	}

	// Backspace edits the query rather than acting as a command.
	next, _ := m.onKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if got := next.(model).filter; got != "daemonse" {
		t.Errorf("after backspace filter = %q, want %q", got, "daemonse")
	}

	// esc abandons the search entirely.
	cleared, _ := next.(model).onKey(tea.KeyMsg{Type: tea.KeyEsc})
	cm := cleared.(model)
	if cm.filtering || cm.filter != "" {
		t.Errorf("esc did not clear the filter (filtering=%v filter=%q)", cm.filtering, cm.filter)
	}
	if len(cm.sel) != 3 {
		t.Errorf("clearing the filter left %d lessons, want all 3 back", len(cm.sel))
	}
}
