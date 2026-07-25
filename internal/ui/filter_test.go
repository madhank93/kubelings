package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/viewport"

	"github.com/madhank93/kubelings/internal/course"
	"github.com/madhank93/kubelings/internal/progress"
)

// modelWithCourse builds a model from real module/lesson data so rebuildRows
// exercises the same path the TUI does.
func modelWithCourse(mods []course.Module) model {
	m := model{w: 100, h: 30}
	m.mods = mods
	m.vp = viewport.New(40, 20)
	m.prog = map[string]progress.State{}
	m.rebuildRows()
	return m
}

func twoModules() []course.Module {
	return []course.Module{
		{Name: "module-01", Title: "Foundations", Lessons: []course.Lesson{
			{Name: "crashloop-triage", Title: "CrashLoopBackOff", ModuleTitle: "Foundations"},
			{Name: "expose-web", Title: "Your first Service", ModuleTitle: "Foundations"},
		}},
		{Name: "module-04", Title: "Networking", Lessons: []course.Lesson{
			{Name: "dns-debug", Title: "The name that won't resolve", ModuleTitle: "Networking"},
		}},
	}
}

func TestFilterMatchesNameTitleAndModule(t *testing.T) {
	cases := []struct {
		filter string
		want   int
		why    string
	}{
		{"", 3, "empty filter shows everything"},
		{"crashloop", 1, "matches lesson name"},
		{"service", 1, "matches lesson title, case-insensitively"},
		{"networking", 1, "matches module title so a module can be isolated"},
		{"e", 3, "substring match, not prefix"},
		{"zzz", 0, "no match yields an empty list, not everything"},
	}
	for _, tc := range cases {
		m := modelWithCourse(twoModules())
		m.filter = tc.filter
		m.rebuildRows()
		if len(m.sel) != tc.want {
			t.Errorf("filter %q: got %d lessons, want %d (%s)", tc.filter, len(m.sel), tc.want, tc.why)
		}
	}
}

// A header with no surviving lesson under it would read as an empty section.
func TestFilterDropsEmptyModuleHeaders(t *testing.T) {
	m := modelWithCourse(twoModules())
	m.filter = "crashloop"
	m.rebuildRows()

	headers := 0
	for _, r := range m.rows {
		if r.header != "" {
			headers++
		}
	}
	if headers != 1 {
		t.Fatalf("got %d module headers, want 1 — only Foundations still has a match", headers)
	}
}

// Narrowing then clearing should not silently move the learner to another lesson.
func TestFilterKeepsSelectionWhenStillVisible(t *testing.T) {
	m := modelWithCourse(twoModules())
	m.cursor = 1 // expose-web
	if got := m.current().Name; got != "expose-web" {
		t.Fatalf("setup: cursor on %q, want expose-web", got)
	}
	m.filter = "e" // matches all three, expose-web included
	m.rebuildRows()
	if got := m.current().Name; got != "expose-web" {
		t.Errorf("after filtering, cursor moved to %q — want it to stay on expose-web", got)
	}
	m.filter = ""
	m.rebuildRows()
	if got := m.current().Name; got != "expose-web" {
		t.Errorf("after clearing the filter, cursor moved to %q — want expose-web", got)
	}
}

func TestFilterClampsCursorWhenSelectionFiltersOut(t *testing.T) {
	m := modelWithCourse(twoModules())
	m.cursor = 2 // dns-debug
	m.filter = "crashloop"
	m.rebuildRows()
	if len(m.sel) != 1 {
		t.Fatalf("got %d lessons, want 1", len(m.sel))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 — it must land inside the narrowed list", m.cursor)
	}
	if m.current() == nil {
		t.Error("current() is nil after filtering; cursor is out of range")
	}
}

func TestNextUnsolvedSkipsSolvedAndWraps(t *testing.T) {
	m := modelWithCourse(twoModules())
	m.prog = map[string]progress.State{
		"expose-web": progress.Solved,
	}

	m.cursor = 0 // crashloop-triage
	if !m.nextUnsolved() {
		t.Fatal("nextUnsolved returned false with unsolved lessons present")
	}
	if got := m.current().Name; got != "dns-debug" {
		t.Errorf("got %q, want dns-debug — expose-web is solved and must be skipped", got)
	}

	// From the last lesson it wraps to the first unsolved one.
	if !m.nextUnsolved() {
		t.Fatal("nextUnsolved returned false when it should wrap")
	}
	if got := m.current().Name; got != "crashloop-triage" {
		t.Errorf("got %q, want crashloop-triage after wrapping", got)
	}
}

func TestNextUnsolvedFalseWhenAllSolved(t *testing.T) {
	m := modelWithCourse(twoModules())
	m.prog = map[string]progress.State{
		"crashloop-triage": progress.Solved,
		"expose-web":       progress.Solved,
		"dns-debug":        progress.Solved,
	}
	if m.nextUnsolved() {
		t.Error("nextUnsolved returned true with every lesson solved")
	}
}

// Guards the modulo in nextUnsolved against an empty (fully filtered) list.
func TestNextUnsolvedEmptyListDoesNotPanic(t *testing.T) {
	m := modelWithCourse(twoModules())
	m.filter = "zzz"
	m.rebuildRows()
	if m.nextUnsolved() {
		t.Error("nextUnsolved returned true on an empty list")
	}
}

func TestRowAtMapsClicksToLessons(t *testing.T) {
	m := modelWithCourse(twoModules())
	top := m.listTop()

	// rows: [header, crashloop, expose-web, header, dns-debug]
	if _, ok := m.rowAt(top); ok {
		t.Error("clicking the module header selected a lesson; it should be inert")
	}
	for _, tc := range []struct {
		y    int
		want string
	}{
		{top + 1, "crashloop-triage"},
		{top + 2, "expose-web"},
		{top + 4, "dns-debug"},
	} {
		i, ok := m.rowAt(tc.y)
		if !ok {
			t.Errorf("y=%d: no lesson hit", tc.y)
			continue
		}
		if got := m.rows[m.sel[i]].lesson.Name; got != tc.want {
			t.Errorf("y=%d: hit %q, want %q", tc.y, got, tc.want)
		}
	}
	if _, ok := m.rowAt(top + 99); ok {
		t.Error("a click past the last row selected something")
	}
	if _, ok := m.rowAt(0); ok {
		t.Error("a click on the header bar selected a lesson")
	}
}

// Clicks must stay correct once the list has scrolled.
func TestRowAtRespectsScrollOffset(t *testing.T) {
	m := modelWithCourse(twoModules())
	m.listOff = 2 // first visible row is expose-web
	i, ok := m.rowAt(m.listTop())
	if !ok {
		t.Fatal("no lesson hit at the top of a scrolled list")
	}
	if got := m.rows[m.sel[i]].lesson.Name; got != "expose-web" {
		t.Errorf("hit %q, want expose-web", got)
	}
}
