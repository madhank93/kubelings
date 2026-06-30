package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/madhank93/kubelings/internal/course"
	"github.com/madhank93/kubelings/internal/progress"
)

var (
	paneStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	headerStyle = lipgloss.NewStyle().Bold(true)
	moduleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	cursorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("63"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	solvedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	startedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	noneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

func leftWidth(w int) int {
	lw := w * 4 / 10
	if lw < 30 {
		lw = 30
	}
	if lw > 48 {
		lw = 48
	}
	if lw > w-12 {
		lw = w - 12
	}
	return lw
}

func (m model) chromeHeight() int {
	banner := 0
	if len(m.issues) > 0 {
		banner = len(m.issues)
	}
	return 1 /*header*/ + banner + 2 /*footer*/ + 2 /*pane border*/
}

func markerStyle(s progress.State) lipgloss.Style {
	switch s {
	case progress.Solved:
		return solvedStyle
	case progress.Started:
		return startedStyle
	default:
		return noneStyle
	}
}

func (m model) headerBar() string {
	st := m.status
	var dot string
	if st.Up {
		dot = okStyle.Render(fmt.Sprintf("● up · %d nodes · %s", st.Nodes, st.Context))
	} else {
		dot = dimStyle.Render("○ down — press u to start")
	}
	return titleStyle.Render("kubelings") + "   cluster: " + dot
}

func (m model) issueBanner() string {
	if len(m.issues) == 0 {
		return ""
	}
	var lines []string
	for _, is := range m.issues {
		lines = append(lines, warnStyle.Render("⚠ "+is.Msg)+dimStyle.Render(" → "+is.Fix))
	}
	return strings.Join(lines, "\n")
}

func (m model) listView() string {
	lw := leftWidth(m.w)
	inner := lw - 4 // border + padding
	bodyH := m.vp.Height
	var lines []string
	for i, r := range m.rows {
		if len(lines) >= bodyH {
			break
		}
		if r.header != "" {
			lines = append(lines, moduleStyle.Render(truncate(r.header, inner)))
			continue
		}
		l := r.lesson
		state := progress.Get(m.prog, l.Name)
		mk := markerStyle(state).Render(state.Marker())
		label := fmt.Sprintf("%s %-16s %s", mk, truncate(l.Name, 16), l.Title)
		label = truncate(label, inner)
		if m.isCursor(i) {
			label = cursorStyle.Render(padRight(label, inner))
		}
		lines = append(lines, label)
	}
	for len(lines) < bodyH {
		lines = append(lines, "")
	}
	content := strings.Join(lines, "\n")
	return paneStyle.Width(lw - 2).Height(bodyH).Render(content)
}

func (m model) isCursor(rowIdx int) bool {
	return len(m.sel) > 0 && m.sel[m.cursor] == rowIdx
}

func (m model) footer() string {
	if m.confirm {
		return warnStyle.Render("reveal solution? y/N")
	}
	keys := helpStyle.Render(
		"i init · v verify · r reset · h hint · s solution · t term · u up · d down · g refresh · ? help · q quit")
	status := ""
	if m.running {
		status = m.spin.View() + " running " + m.runLbl + "…"
	}
	return keys + "\n" + status
}

// refreshView sets the right-pane content for the current mode + lesson.
func (m *model) refreshView() {
	l := m.current()
	switch m.mode {
	case modeHelp:
		m.vp.SetContent(helpText())
	case modeHint:
		if l != nil && l.Hint != "" {
			m.vp.SetContent(headerStyle.Render("Hint — "+l.Title) + "\n\n" + l.Hint)
		} else {
			m.vp.SetContent("(no hint for this lesson)")
		}
		m.vp.GotoTop()
	case modeSolution:
		if l != nil && l.Solution != "" {
			m.vp.SetContent(headerStyle.Render("Solution — "+l.Title) + "\n\n" + l.Solution)
		} else {
			m.vp.SetContent("(no solution for this lesson)")
		}
		m.vp.GotoTop()
	case modeOutput:
		// content already set by the action handler
	default: // modeDetail
		m.vp.SetContent(m.detail(l))
		m.vp.GotoTop()
	}
}

func (m model) detail(l *course.Lesson) string {
	if l == nil {
		return "no lessons found under courses/kubelings/"
	}
	state := progress.Get(m.prog, l.Name)
	var b strings.Builder
	b.WriteString(titleStyle.Render(l.Title) + "\n\n")
	b.WriteString(l.Description + "\n\n")
	b.WriteString(dimStyle.Render("lesson:     ") + l.Name + "\n")
	b.WriteString(dimStyle.Render("playground: ") + l.Playground + "\n")
	b.WriteString(dimStyle.Render("status:     ") + markerStyle(state).Render(string(state)+" "+state.Marker()) + "\n\n")
	b.WriteString(helpStyle.Render("i init · v verify · h hint · s solution · t shell"))
	return b.String()
}

func helpText() string {
	return headerStyle.Render("Keys") + "\n\n" +
		"  ↑/k ↓/j   navigate lessons\n" +
		"  i / enter init the selected scenario (build it)\n" +
		"  v         verify your fix\n" +
		"  r         reset (wipe ns + re-init)\n" +
		"  h         show hint\n" +
		"  s         show solution (asks to confirm)\n" +
		"  t         drop into a shell wired to the cluster\n" +
		"  u / d     cluster up / down\n" +
		"  g         refresh status & progress\n" +
		"  esc       back to lesson detail\n" +
		"  ? / q     toggle help / quit\n\n" +
		dimStyle.Render("markers: ◌ not started · ◐ started · ✓ solved")
}

// course import alias for detail() signature.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	// account for embedded ANSI roughly: only truncate plain content
	if lipgloss.Width(s) <= w {
		return s
	}
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

func padRight(s string, w int) string {
	d := w - lipgloss.Width(s)
	if d <= 0 {
		return s
	}
	return s + strings.Repeat(" ", d)
}
