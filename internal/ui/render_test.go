package ui

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/madhank93/kubelings/internal/course"
	"github.com/madhank93/kubelings/internal/runner"
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

// Cluster facts are identical for every lesson, so they belong in the header
// bar — repeating them per lesson pushed the scenario text off the pane.
func TestHeaderCarriesClusterFactsAndDetailDoesNot(t *testing.T) {
	m := renderModel()
	m.w = 120
	m.status = runner.ClusterStatus{Up: true, Context: "kind-kubelings", Nodes: 3, Version: "v1.31.0"}

	h := plain(m.headerBar())
	for _, want := range []string{"kind-kubelings", "3 nodes", "v1.31.0", "ns kubelings", "persists across lessons"} {
		if !strings.Contains(h, want) {
			t.Errorf("header bar is missing %q:\n%s", want, h)
		}
	}

	l := &course.Lesson{Name: "dns-debug", Title: "DNS", HasTasks: true, Playground: "k8s-omni"}
	d := plain(m.detail(l))
	for _, gone := range []string{"Cluster", "namespace:", "lifecycle:", "3 nodes"} {
		if strings.Contains(d, gone) {
			t.Errorf("detail still repeats header/cluster info %q:\n%s", gone, d)
		}
	}
	// The playground name is iximiuz plumbing, not something the learner acts
	// on — it belongs to the cloud-only block, not to every lesson.
	if strings.Contains(d, "k8s-omni") {
		t.Errorf("detail still prints the playground name:\n%s", d)
	}
}

// The footer already lists ↵/i/v/h/s/t on every frame; printing them again in
// the detail pane was pure duplication.
func TestDetailOmitsKeysTheFooterAlreadyShows(t *testing.T) {
	m := renderModel()
	d := plain(m.detail(&course.Lesson{Name: "dns-debug", Title: "DNS", HasTasks: true}))
	for _, gone := range []string{"play (cluster", "i init", "v verify", "t shell"} {
		if strings.Contains(d, gone) {
			t.Errorf("detail duplicates footer keybar %q:\n%s", gone, d)
		}
	}
}

// Mouse capture swallows drags, so the terminal cannot select text. `m` must
// release it (and say so) or copy/paste is impossible inside the TUI.
func TestCopyModeReleasesMouseAndAnnouncesItself(t *testing.T) {
	m := renderModel()
	if m.mouseOff {
		t.Fatal("mouse capture should start enabled")
	}
	upd, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m = upd.(model)
	if !m.mouseOff {
		t.Fatal("m did not enter copy mode")
	}
	if cmd == nil {
		t.Fatal("entering copy mode issued no command; mouse is still captured")
	}
	if got, want := reflect.TypeOf(cmd()), reflect.TypeOf(tea.DisableMouse()); got != want {
		t.Errorf("copy mode sent %v, want %v", got, want)
	}
	if f := plain(m.footer()); !strings.Contains(f, "copy mode") {
		t.Errorf("copy mode is invisible in the footer:\n%s", f)
	}

	upd, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	if upd.(model).mouseOff {
		t.Error("second m did not restore mouse capture")
	}
	if cmd == nil {
		t.Error("leaving copy mode issued no command; wheel scroll stays dead")
	}
}

// On a narrow terminal the header must stay on one line — the layout counts it
// as exactly one row (listTop/chromeHeight), so a wrap would shift every
// mouse hit-test and clip the panes.
func TestHeaderFitsNarrowTerminals(t *testing.T) {
	for _, w := range []int{80, 100, 120} {
		m := renderModel()
		m.w = w
		m.status = runner.ClusterStatus{Up: true, Context: "kind-kubelings", Nodes: 3, Version: "v1.31.0"}
		if got := lipgloss.Width(m.headerBar()); got > w {
			t.Errorf("header is %d cells wide at w=%d; it will wrap:\n%s", got, w, plain(m.headerBar()))
		}
		m.status = runner.ClusterStatus{Up: false}
		if got := lipgloss.Width(m.headerBar()); got > w {
			t.Errorf("down-state header is %d cells wide at w=%d:\n%s", got, w, plain(m.headerBar()))
		}
	}
}

// Lesson diagrams are box art in a ```text fence. Glamour word-wraps prose to
// the pane width; if it ever wraps a fenced block too, the art turns to rubble.
func TestDiagramFenceSurvivesRendering(t *testing.T) {
	art, err := os.ReadFile("../../courses/kubelings/module-01/04.selector-mismatch/diagrams/topology.txt")
	if err != nil {
		t.Skipf("no diagram to check: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(art), "\n"), "\n")
	md := "## The situation\n\nEvery request to the `api` Service times out even though the pods are healthy.\n\n```text\n" +
		strings.Join(lines, "\n") + "\n```\n"

	// 44 is narrower than the pane at an 80-column terminal — if anything wraps,
	// it wraps here first.
	out := renderMarkdown(md, 44)
	for _, want := range lines {
		if strings.TrimSpace(want) == "" {
			continue
		}
		if !strings.Contains(plain(out), strings.TrimRight(want, " ")) {
			t.Fatalf("diagram line was mangled by the renderer:\nwant %q\ngot:\n%s", want, plain(out))
		}
	}
}

// The detail pane must show the lesson prose, not just its metadata — the body
// is the lesson, and it used to be visible only inside the shell.
func TestDetailRendersLessonBody(t *testing.T) {
	m := renderModel()
	m.vp = viewport.New(56, 20)
	l := &course.Lesson{
		Name: "dns-debug", Title: "DNS", HasTasks: true,
		Description: "The `api` Service exists.",
		Task:        "## The situation\n\nEndpoints are **empty**.\n",
	}
	d := plain(m.detail(l))
	if !strings.Contains(d, "The situation") || !strings.Contains(d, "Endpoints are") {
		t.Errorf("detail pane does not show the lesson body:\n%s", d)
	}
	// Markdown is rendered, not dumped: no literal ## or ** survive.
	if strings.Contains(d, "##") || strings.Contains(d, "**") {
		t.Errorf("body is raw markdown, not rendered:\n%s", d)
	}
	if strings.Contains(d, "`api`") {
		t.Errorf("description is raw markdown, not rendered:\n%s", d)
	}
	// Second call must come from the cache, not a second glamour run.
	if len(m.mdCache) == 0 {
		t.Error("render cache is empty; every cursor move will re-render")
	}
}

// A diagram sits in the middle of the pane, not shoved against the left edge
// where glamour's code-block rendering would leave it.
func TestDiagramIsCenteredInThePane(t *testing.T) {
	art, err := os.ReadFile("../../courses/kubelings/module-01/04.selector-mismatch/diagrams/topology.txt")
	if err != nil {
		t.Skipf("no diagram to check: %v", err)
	}
	body := "Endpoints are empty.\n\n<!-- d2:topology -->\n```text\n" +
		strings.TrimRight(string(art), "\n") + "\n```\n<!-- /d2:topology -->\n\n## Your task\n\n1. Fix it.\n"

	const w = 56
	out := plain(renderLesson(body, w))
	if strings.Contains(out, "d2:topology") {
		t.Errorf("generator markers leaked into the render:\n%s", out)
	}
	// The art is centered as one block, so measure the block: D2's own rows are
	// padded and inset relative to each other, and no single row is centered.
	lines := strings.Split(strings.TrimRight(string(art), "\n"), "\n")
	blockW := 0
	for _, l := range lines {
		if x := lipgloss.Width(strings.TrimRight(l, " ")); x > blockW {
			blockW = x
		}
	}
	pad := strings.Repeat(" ", (w-blockW)/2)
	for _, l := range lines {
		l = strings.TrimRight(l, " ")
		if strings.TrimSpace(l) == "" {
			continue
		}
		if !strings.Contains(out, pad+l) {
			t.Fatalf("diagram row is not centered (expected %d columns of left pad):\nrow %q\ngot:\n%s",
				len(pad), l, out)
		}
	}
}
