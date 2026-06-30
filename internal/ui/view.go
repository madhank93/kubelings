package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/madhank93/kubelings/internal/course"
	"github.com/madhank93/kubelings/internal/progress"
)

var (
	paneStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("60")).Padding(0, 1)
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("153"))
	moduleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
	cursorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("63"))
	textStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	warnStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))

	keyStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("87"))
	sepStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	solvedStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	startedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	noneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

// keybar renders "key label · key label" with colored keys.
func keybar(pairs ...[2]string) string {
	var parts []string
	for _, p := range pairs {
		parts = append(parts, keyStyle.Render(p[0])+" "+textStyle.Render(p[1]))
	}
	return strings.Join(parts, sepStyle.Render(" · "))
}

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
		name := truncate(l.Name, inner-2)
		if m.isCursor(i) {
			lines = append(lines, cursorStyle.Render(padRight(" "+mk+" "+name, inner)))
		} else {
			lines = append(lines, " "+mk+" "+textStyle.Render(name))
		}
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
	if m.confirmSwitch {
		t := ""
		if m.switchTarget != nil {
			t = m.switchTarget.Name
		}
		return warnStyle.Render("⚠ scenario '"+m.switchOther+"' still active — ") +
			keybar([2]string{"d", "destroy & start " + t}, [2]string{"k", "keep & start"}, [2]string{"c", "cancel"})
	}
	if m.confirm {
		return warnStyle.Render("reveal solution? ") + keybar([2]string{"y", "yes"}, [2]string{"N", "no"})
	}
	keys := keybar(
		[2]string{"↵", "play"}, [2]string{"i", "init"}, [2]string{"v", "verify"}, [2]string{"r", "reset"},
		[2]string{"h", "hint"}, [2]string{"s", "solution"}, [2]string{"t", "shell"},
		[2]string{"u", "up"}, [2]string{"d", "down"}, [2]string{"g", "refresh"}, [2]string{"?", "help"}, [2]string{"q", "quit"})
	status := ""
	if m.running {
		status = startedStyle.Render(m.spin.View()+" running "+m.runLbl) + dimStyle.Render(" …")
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
	b.WriteString(textStyle.Render(l.Description) + "\n\n")
	b.WriteString(dimStyle.Render("lesson:     ") + textStyle.Render(l.Name) + "\n")
	b.WriteString(dimStyle.Render("playground: ") + textStyle.Render(l.Playground) + "\n")
	b.WriteString(dimStyle.Render("status:     ") + markerStyle(state).Render(state.Marker()+" "+string(state)) + "\n\n")
	b.WriteString(keybar([2]string{"↵", "play (cluster + init + shell)"}) + "\n")
	b.WriteString(keybar([2]string{"i", "init"}, [2]string{"v", "verify"}, [2]string{"h", "hint"}, [2]string{"s", "solution"}, [2]string{"t", "shell"}))
	return b.String()
}

func helpText() string {
	row := func(k, d string) string { return "  " + keyStyle.Render(padRight(k, 9)) + textStyle.Render(d) + "\n" }
	return headerStyle.Render("Keys") + "\n\n" +
		row("↵ / space", "play — cluster up (if needed) + init + open shell") +
		row("i", "init the selected scenario (build it)") +
		row("v", "verify your fix") +
		row("r", "reset (wipe ns + re-init)") +
		row("h", "show hint") +
		row("s", "show solution (asks to confirm)") +
		row("t", "drop into a shell wired to the cluster") +
		row("u / d", "cluster up / down") +
		row("g", "refresh status & progress") +
		row("esc", "back to lesson detail") +
		row("? / q", "toggle help / quit") + "\n" +
		dimStyle.Render("markers: ") + noneStyle.Render("◌ not started") + dimStyle.Render(" · ") +
		startedStyle.Render("◐ started") + dimStyle.Render(" · ") + solvedStyle.Render("✓ solved") + "\n\n" +
		dimStyle.Render("in the shell: ") + keyStyle.Render("task") + dimStyle.Render(" · ") +
		keyStyle.Render("hint") + dimStyle.Render(" · ") + keyStyle.Render("verify") + dimStyle.Render(" · ") +
		keyStyle.Render("solution") + dimStyle.Render(" · ") + keyStyle.Render("k=kubectl")
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
