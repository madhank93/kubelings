package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"

	"github.com/madhank93/kubelings/internal/course"
)

// buildModel makes a model with n modules × k lessons and a list pane h rows tall.
func buildModel(nMods, nLessons, h int) model {
	m := model{w: 100, h: h}
	for mo := 0; mo < nMods; mo++ {
		m.rows = append(m.rows, row{header: "Module"})
		for i := 0; i < nLessons; i++ {
			l := &course.Lesson{Name: "lesson", Title: "t"}
			m.rows = append(m.rows, row{lesson: l})
			m.sel = append(m.sel, len(m.rows)-1)
		}
	}
	m.vp = viewport.New(40, h)
	return m
}

func TestListScrollKeepsCursorVisible(t *testing.T) {
	m := buildModel(9, 6, 10) // 63 rows, 10 visible
	for c := range m.sel {
		m.cursor = c
		m.clampListOff()
		cur := m.sel[c]
		if cur < m.listOff || cur >= m.listOff+m.vp.Height {
			t.Fatalf("cursor row %d not visible in window [%d,%d)", cur, m.listOff, m.listOff+m.vp.Height)
		}
	}
	// scroll back up to the top
	m.cursor = 0
	m.clampListOff()
	if m.listOff != 0 {
		t.Fatalf("expected offset 0 at first lesson (header pulled along), got %d", m.listOff)
	}
}

func TestListScrollHeaderPulledAlong(t *testing.T) {
	m := buildModel(9, 6, 10)
	// cursor on the first lesson of module 5 (row after its header)
	for c, ri := range m.sel {
		if ri > 30 && m.rows[ri-1].header != "" {
			m.cursor = c
			break
		}
	}
	m.listOff = len(m.rows) // force clamp from far below
	m.clampListOff()
	cur := m.sel[m.cursor]
	if m.listOff > cur-1 {
		t.Fatalf("module header row %d scrolled out: offset %d", cur-1, m.listOff)
	}
}

func TestListViewHeightStable(t *testing.T) {
	m := buildModel(9, 6, 12)
	for c := range m.sel {
		m.cursor = c
		m.clampListOff()
		v := m.listView()
		if got := strings.Count(v, "\n") + 1; got != 12+2 { // body + top/bottom border
			t.Fatalf("cursor %d: pane is %d lines, want %d", c, got, 14)
		}
	}
}
