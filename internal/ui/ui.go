// Package ui is the bubbletea TUI. It is UI-only: every cluster/lesson action is
// delegated to the bash runner (internal/runner). The course (internal/course) is
// the source of truth it renders; progress markers come from internal/progress.
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/madhank93/kubelings/internal/course"
	"github.com/madhank93/kubelings/internal/preflight"
	"github.com/madhank93/kubelings/internal/progress"
	"github.com/madhank93/kubelings/internal/runner"
)

type viewMode int

const (
	modeDetail viewMode = iota
	modeOutput
	modeHint
	modeSolution
	modeHelp
)

type row struct {
	header string         // non-empty => module header (not selectable)
	lesson *course.Lesson // non-nil => lesson row
}

type model struct {
	root    string
	rows    []row
	sel     []int // indices into rows that are lessons
	cursor  int
	prog    map[string]progress.State
	status  runner.ClusterStatus
	issues  []preflight.Issue
	vp      viewport.Model
	spin    spinner.Model
	mode    viewMode
	running bool
	runLbl  string
	confirm bool // solution reveal prompt
	w, h    int
	ready   bool
}

type runDoneMsg struct {
	action, lesson, out string
	ok                  bool
}
type execDoneMsg struct{}

// New builds the initial model.
func New(root string) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	m := model{root: root, spin: sp, mode: modeDetail}
	m.reload()
	return m
}

// reload rebuilds course rows + progress + status + preflight.
func (m *model) reload() {
	mods, _ := course.Discover(m.root)
	m.rows = nil
	m.sel = nil
	for _, mo := range mods {
		title := mo.Title
		if title == "" {
			title = mo.Name
		}
		m.rows = append(m.rows, row{header: title})
		for i := range mo.Lessons {
			l := mo.Lessons[i]
			m.rows = append(m.rows, row{lesson: &l})
			m.sel = append(m.sel, len(m.rows)-1)
		}
	}
	if m.cursor >= len(m.sel) {
		m.cursor = max(0, len(m.sel)-1)
	}
	m.prog = progress.Load(m.root)
	m.status = runner.Status()
	m.issues = preflight.Check()
}

func (m model) current() *course.Lesson {
	if len(m.sel) == 0 {
		return nil
	}
	return m.rows[m.sel[m.cursor]].lesson
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.layout()
		m.ready = true
		m.refreshView()
		return m, nil

	case spinner.TickMsg:
		if !m.running {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case runDoneMsg:
		m.running = false
		m.prog = progress.Load(m.root)
		m.status = runner.Status()
		m.mode = modeOutput
		head := fmt.Sprintf("$ %s %s\n\n", msg.lesson, msg.action)
		m.vp.SetContent(head + msg.out)
		m.vp.GotoTop()
		return m, nil

	case execDoneMsg:
		m.reload()
		m.refreshView()
		return m, nil

	case tea.KeyMsg:
		return m.onKey(msg)
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m model) onKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Solution reveal confirmation captures keys first.
	if m.confirm {
		switch msg.String() {
		case "y", "Y":
			m.confirm = false
			m.mode = modeSolution
			m.refreshView()
		case "n", "N", "esc", "q":
			m.confirm = false
			m.refreshView()
		}
		return m, nil
	}
	if m.running {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.mode = modeDetail
		m.refreshView()
	case "down", "j":
		if m.cursor < len(m.sel)-1 {
			m.cursor++
		}
		m.mode = modeDetail
		m.refreshView()
	case "esc":
		m.mode = modeDetail
		m.refreshView()
	case "g":
		m.reload()
		m.refreshView()
	case "?":
		if m.mode == modeHelp {
			m.mode = modeDetail
		} else {
			m.mode = modeHelp
		}
		m.refreshView()
	case "h":
		m.mode = modeHint
		m.refreshView()
	case "s":
		if l := m.current(); l != nil && l.Solution != "" {
			m.confirm = true
		}
	case "i", "enter":
		return m.runAction("init")
	case "v":
		return m.runAction("verify")
	case "r":
		return m.runAction("reset")
	case "u":
		return m, tea.ExecProcess(runner.Cmd(m.root, "up"), func(error) tea.Msg { return execDoneMsg{} })
	case "d":
		return m, tea.ExecProcess(runner.Cmd(m.root, "down"), func(error) tea.Msg { return execDoneMsg{} })
	case "t":
		return m.openShell()
	case "pgup", "pgdown", "ctrl+u", "ctrl+d":
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

// runAction launches a lesson action via the runner, with a cluster pre-check.
func (m model) runAction(action string) (tea.Model, tea.Cmd) {
	l := m.current()
	if l == nil {
		return m, nil
	}
	if !m.status.Up {
		m.mode = modeOutput
		m.vp.SetContent("cluster not up — press u to start it.")
		return m, nil
	}
	m.running = true
	m.runLbl = action + " " + l.Name
	root, name := m.root, l.Name
	return m, tea.Batch(
		m.spin.Tick,
		func() tea.Msg {
			out, ok := runner.Capture(root, name, action)
			return runDoneMsg{action: action, lesson: name, out: out, ok: ok}
		},
	)
}

// openShell drops into an interactive shell wired to the kind cluster, isolated
// via a temp KUBECONFIG so the user's global context is untouched.
func (m model) openShell() (tea.Model, tea.Cmd) {
	if !m.status.Up {
		m.mode = modeOutput
		m.vp.SetContent("cluster not up — press u to start it before opening a shell.")
		return m, nil
	}
	kubeconfig, rc, err := writeShellEnv(m.status.Context)
	if err != nil {
		m.mode = modeOutput
		m.vp.SetContent("could not prepare shell: " + err.Error())
		return m, nil
	}
	c := exec.Command("bash", "--rcfile", rc, "-i")
	c.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	return m, tea.ExecProcess(c, func(error) tea.Msg { return execDoneMsg{} })
}

func (m *model) layout() {
	left := leftWidth(m.w)
	bodyH := m.h - m.chromeHeight()
	if bodyH < 3 {
		bodyH = 3
	}
	rightW := m.w - left - 3
	if rightW < 10 {
		rightW = 10
	}
	m.vp = viewport.New(rightW, bodyH)
}

func (m model) View() string {
	if !m.ready {
		return "loading…"
	}
	var b strings.Builder
	b.WriteString(m.headerBar() + "\n")
	if banner := m.issueBanner(); banner != "" {
		b.WriteString(banner + "\n")
	}
	left := m.listView()
	right := paneStyle.Render(m.vp.View())
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, right) + "\n")
	b.WriteString(m.footer())
	return b.String()
}
