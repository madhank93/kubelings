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
	listOff int // first visible row in the left list (scroll offset)
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
	splash  bool // show the welcome splash

	// play / shell chaining + switch-scenario guard
	pendingPlay   *course.Lesson // run init+shell after the cluster comes up
	openShellNext bool           // after the current init/reset finishes, drop into the shell
	shellLesson   *course.Lesson
	lastAction    string
	confirmSwitch bool
	switchTarget  *course.Lesson
	switchOther   string
	switchShell   bool
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
	m := model{root: root, spin: sp, mode: modeDetail, splash: true}
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
	m.clampListOff()
}

// clampListOff scrolls the left list just enough to keep the cursor visible,
// pulling the module header along when the cursor sits right under it.
func (m *model) clampListOff() {
	bodyH := m.vp.Height
	if bodyH <= 0 || len(m.sel) == 0 {
		m.listOff = 0
		return
	}
	cur := m.sel[m.cursor]
	top := cur
	if top > 0 && m.rows[top-1].header != "" {
		top--
	}
	if top < m.listOff {
		m.listOff = top
	}
	if cur >= m.listOff+bodyH {
		m.listOff = cur - bodyH + 1
	}
	m.listOff = min(m.listOff, max(0, len(m.rows)-bodyH))
	m.listOff = max(m.listOff, 0)
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
		// init/reset followed by a requested shell drop?
		if m.openShellNext && msg.ok && (msg.action == "init" || msg.action == "reset") {
			m.openShellNext = false
			l := m.shellLesson
			m.mode = modeDetail
			m.refreshView()
			cmd := m.execShell(l)
			return m, cmd
		}
		m.openShellNext = false
		m.mode = modeOutput
		m.vp.SetContent(fmt.Sprintf("$ %s %s\n\n", msg.lesson, msg.action) + msg.out)
		m.vp.GotoTop()
		return m, nil

	case execDoneMsg:
		m.reload()
		// finishing a "play": cluster is up, now init+shell the pending lesson.
		if m.pendingPlay != nil && m.status.Up {
			l := m.pendingPlay
			m.pendingPlay = nil
			return m.beginInit(l, true)
		}
		m.pendingPlay = nil
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
	// Splash: any key dismisses (q/ctrl+c still quits).
	if m.splash {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		default:
			m.splash = false
			m.refreshView()
			return m, nil
		}
	}
	// Switch-scenario guard captures keys first.
	if m.confirmSwitch {
		switch msg.String() {
		case "d", "D": // destroy current, then start target
			m.confirmSwitch = false
			for _, o := range progress.StartedLessons(m.prog) {
				if o != m.switchTarget.Name {
					_ = progress.Set(m.root, o, progress.None)
				}
			}
			m.prog = progress.Load(m.root)
			return m.launchInit(m.switchTarget, m.switchShell, true /*reset*/)
		case "k", "K": // keep current resources, init target over it
			m.confirmSwitch = false
			return m.launchInit(m.switchTarget, m.switchShell, false)
		case "c", "C", "esc", "n", "N":
			m.confirmSwitch = false
		}
		return m, nil
	}
	// Solution reveal confirmation.
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
		m.clampListOff()
		m.mode = modeDetail
		m.refreshView()
	case "down", "j":
		if m.cursor < len(m.sel)-1 {
			m.cursor++
		}
		m.clampListOff()
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
	case "a":
		m.splash = true
	case "h":
		m.mode = modeHint
		m.refreshView()
	case "s":
		if l := m.current(); l != nil && l.Solution != "" {
			m.confirm = true
		}
	case "enter", " ": // PLAY: cluster up (if needed) -> init -> shell
		l := m.current()
		if l == nil {
			return m, nil
		}
		// Readings have nothing to run; cloud-only lessons have nothing to run
		// *here*. Both self-attest with ↵ instead of playing.
		if !l.HasTasks || l.CloudOnly {
			s := progress.Solved
			if progress.Get(m.prog, l.Name) == progress.Solved {
				s = progress.None
			}
			_ = progress.Set(m.root, l.Name, s)
			m.prog = progress.Load(m.root)
			m.refreshView()
			return m, nil
		}
		if !m.status.Up {
			m.pendingPlay = l
			return m, tea.ExecProcess(runner.Cmd(m.root, "up"), func(error) tea.Msg { return execDoneMsg{} })
		}
		return m.beginInit(l, true)
	case "i":
		if l := m.current(); l != nil && l.HasTasks {
			if m.blockCloudOnly(l) {
				return m, nil
			}
			return m.beginInit(l, false)
		}
	case "v":
		return m.runAction("verify")
	case "r":
		return m.runAction("reset")
	case "u":
		return m, tea.ExecProcess(runner.Cmd(m.root, "up"), func(error) tea.Msg { return execDoneMsg{} })
	case "d":
		return m, tea.ExecProcess(runner.Cmd(m.root, "down"), func(error) tea.Msg { return execDoneMsg{} })
	case "t":
		if l := m.current(); l != nil {
			// The shell's rcfile wires a `verify` helper straight to the local
			// runner, so the shell is a local-execution door like any other.
			if m.blockCloudOnly(l) {
				return m, nil
			}
			cmd := m.execShell(l)
			return m, cmd
		}
	case "pgup", "pgdown", "ctrl+u", "ctrl+d":
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

// blockCloudOnly refuses a local-execution action for a cloud-only lesson and
// renders why. Every door into the local runner calls it — the key handlers for
// the affordance, and runAction/beginInit/execShell as a structural backstop so
// a future key path can't bypass the gate.
func (m *model) blockCloudOnly(l *course.Lesson) bool {
	if l == nil || !l.CloudOnly {
		return false
	}
	reason := l.CloudOnlyReason
	if reason == "" {
		reason = "it needs real-VM/host access"
	}
	// Reasons are full sentences; wrap them rather than letting the viewport
	// clip the end of the explanation.
	wrap := textStyle
	if w := m.vp.Width; w > 20 {
		wrap = wrap.Width(w)
	}
	m.mode = modeOutput
	m.vp.SetContent(warnStyle.Render("☁ ‘"+l.Title+"’ runs on iximiuz Labs only.") + "\n\n" +
		wrap.Render("It can't run on your local kind cluster because "+reason+".") + "\n" +
		dimStyle.Render("Lesson scripts are confined to the kind node container, so host-level\n"+
			"work has nowhere to happen locally. On iximiuz it runs on disposable VMs.") + "\n\n" +
		dimStyle.Render("Run it here: ") + linkStyle.Render(course.CourseURL(m.root)) + "\n\n" +
		keybar([2]string{"↵", "mark done / not done"}))
	m.vp.GotoTop()
	return true
}

// beginInit starts a lesson, prompting first if a different scenario is still active.
func (m model) beginInit(l *course.Lesson, withShell bool) (tea.Model, tea.Cmd) {
	if m.blockCloudOnly(l) {
		return m, nil
	}
	if !m.status.Up {
		m.mode = modeOutput
		m.vp.SetContent("cluster not up — press u (or enter to play).")
		return m, nil
	}
	for _, o := range progress.StartedLessons(m.prog) {
		if o != l.Name {
			m.confirmSwitch = true
			m.switchTarget = l
			m.switchOther = o
			m.switchShell = withShell
			return m, nil
		}
	}
	return m.launchInit(l, withShell, false)
}

// launchInit runs init (or reset) for a lesson, optionally chaining into the shell.
func (m model) launchInit(l *course.Lesson, withShell, reset bool) (tea.Model, tea.Cmd) {
	action := "init"
	if reset {
		action = "reset"
	}
	m.openShellNext = withShell
	m.shellLesson = l
	return m.runAction(action)
}

// runAction launches a lesson action via the runner, with a cluster pre-check.
func (m model) runAction(action string) (tea.Model, tea.Cmd) {
	l := m.current()
	if m.shellLesson != nil && (action == "init" || action == "reset") {
		l = m.shellLesson
	}
	if l == nil {
		return m, nil
	}
	// A reading has no tasks: the runner would print "(content-only lesson)",
	// exit 0, and the caller would record it as solved. Refuse here so no key
	// path can mark a reading solved by "verifying" nothing.
	if !l.HasTasks {
		m.mode = modeOutput
		m.vp.SetContent("‘" + l.Title + "’ is a reading — nothing to " + action + ".\n\nPress ↵ to mark it read / unread.")
		return m, nil
	}
	if m.blockCloudOnly(l) {
		return m, nil
	}
	if !m.status.Up {
		m.mode = modeOutput
		m.vp.SetContent("cluster not up — press u to start it.")
		return m, nil
	}
	m.running = true
	m.runLbl = action + " " + l.Name
	m.lastAction = action
	root, name := m.root, l.Name
	return m, tea.Batch(
		m.spin.Tick,
		func() tea.Msg {
			out, ok := runner.Capture(root, name, action)
			return runDoneMsg{action: action, lesson: name, out: out, ok: ok}
		},
	)
}

// execShell drops into an interactive shell wired to the cluster, showing the
// lesson task and exposing task/hint/verify/solution commands. Isolated via a
// temp KUBECONFIG so the user's global context is untouched.
func (m *model) execShell(l *course.Lesson) tea.Cmd {
	if m.blockCloudOnly(l) {
		return nil
	}
	if !m.status.Up {
		m.mode = modeOutput
		m.vp.SetContent("cluster not up — press u to start it before opening a shell.")
		return nil
	}
	kubeconfig, rc, err := shellEnv(m.root, l)
	if err != nil {
		m.mode = modeOutput
		m.vp.SetContent("could not prepare shell: " + err.Error())
		return nil
	}
	c := exec.Command("bash", "--rcfile", rc, "-i")
	c.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	return tea.ExecProcess(c, func(error) tea.Msg { return execDoneMsg{} })
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
	m.clampListOff()
}

func (m model) View() string {
	if !m.ready {
		return "loading…"
	}
	if m.splash {
		return m.splashView()
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
