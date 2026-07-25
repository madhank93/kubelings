// Package ui is the bubbletea TUI. It is UI-only: every cluster/lesson action is
// delegated to the bash runner (internal/runner). The course (internal/course) is
// the source of truth it renders; progress markers come from internal/progress.
package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	mods    []course.Module // everything discovered, before filtering
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

	// incremental filter over the lesson list ("/" to open)
	filter    string
	filtering bool

	confirmDown bool               // guard on the destroy-cluster key
	cancelRun   context.CancelFunc // set while an action is in flight
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

// reload re-reads the course from disk, then rebuilds rows + progress + status.
func (m *model) reload() {
	m.mods, _ = course.Discover(m.root)
	m.prog = progress.Load(m.root)
	m.status = runner.Status()
	m.issues = preflight.Check()
	m.rebuildRows()
}

// rebuildRows derives the visible list from m.mods and the active filter. A
// module header only appears when it still has a matching lesson under it, so a
// filter never leaves a bare heading with nothing beneath it.
func (m *model) rebuildRows() {
	// Keep the selected lesson selected across a filter edit where possible —
	// re-indexing under the cursor is disorienting.
	var want string
	if l := m.current(); l != nil {
		want = l.Name
	}

	m.rows = nil
	m.sel = nil
	for _, mo := range m.mods {
		title := mo.Title
		if title == "" {
			title = mo.Name
		}
		hdr := -1
		for i := range mo.Lessons {
			l := mo.Lessons[i]
			if !m.matches(&l) {
				continue
			}
			if hdr == -1 {
				m.rows = append(m.rows, row{header: title})
				hdr = len(m.rows) - 1
			}
			m.rows = append(m.rows, row{lesson: &l})
			m.sel = append(m.sel, len(m.rows)-1)
		}
	}

	m.cursor = 0
	if want != "" {
		for i, ri := range m.sel {
			if m.rows[ri].lesson.Name == want {
				m.cursor = i
				break
			}
		}
	}
	if m.cursor >= len(m.sel) {
		m.cursor = max(0, len(m.sel)-1)
	}
	m.clampListOff()
}

// matches reports whether a lesson survives the active filter. Matching covers
// name, title and module title, so "net" finds the Networking module and
// "crashloop" finds the lesson, without the learner knowing which is which.
func (m model) matches(l *course.Lesson) bool {
	if m.filter == "" {
		return true
	}
	q := strings.ToLower(m.filter)
	return strings.Contains(strings.ToLower(l.Name), q) ||
		strings.Contains(strings.ToLower(l.Title), q) ||
		strings.Contains(strings.ToLower(l.ModuleTitle), q)
}

// nextUnsolved moves the cursor to the next lesson that isn't solved, wrapping
// once. With 117 lessons, "where was I" should not be a scrolling exercise.
func (m *model) nextUnsolved() bool {
	n := len(m.sel)
	if n == 0 {
		return false
	}
	for i := 1; i <= n; i++ {
		c := (m.cursor + i) % n
		l := m.rows[m.sel[c]].lesson
		if progress.Get(m.prog, l.Name) != progress.Solved {
			m.cursor = c
			m.clampListOff()
			return true
		}
	}
	return false
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

// rowAt maps a terminal row to an index into m.sel, or false when the click
// landed on a module header, on padding, or outside the list body.
func (m model) rowAt(y int) (int, bool) {
	i := m.listOff + y - m.listTop()
	if i < m.listOff || i >= len(m.rows) || m.rows[i].lesson == nil {
		return 0, false
	}
	for si, ri := range m.sel {
		if ri == i {
			return si, true
		}
	}
	return 0, false
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
		if m.cancelRun != nil {
			m.cancelRun() // release the context; the process has already exited
			m.cancelRun = nil
		}
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

	case tea.MouseMsg:
		return m.onMouse(msg)
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// onMouse handles the scroll wheel: over the left list it moves the selection
// (like j/k, which the list couples to its scroll offset); over the right pane
// it scrolls the viewport. Mouse mode is enabled in main.go.
func (m model) onMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Modal states stay keyboard-only.
	if m.splash || m.confirm || m.confirmSwitch || m.confirmDown || m.running {
		return m, nil
	}
	overList := msg.X < leftWidth(m.w)
	switch msg.Button {
	case tea.MouseButtonLeft:
		// Wheel-without-click reads as a half-broken list: the pointer moves the
		// selection but pressing on a row does nothing. Make the click land.
		if overList && msg.Action == tea.MouseActionPress {
			if i, ok := m.rowAt(msg.Y); ok {
				m.cursor = i
				m.clampListOff()
				m.mode = modeDetail
				m.refreshView()
			}
		}
		return m, nil
	case tea.MouseButtonWheelUp:
		if overList {
			if m.cursor > 0 {
				m.cursor--
			}
			m.clampListOff()
			m.mode = modeDetail
			m.refreshView()
			return m, nil
		}
	case tea.MouseButtonWheelDown:
		if overList {
			if m.cursor < len(m.sel)-1 {
				m.cursor++
			}
			m.clampListOff()
			m.mode = modeDetail
			m.refreshView()
			return m, nil
		}
	}
	// Right pane (or a horizontal wheel): let the viewport scroll.
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
	// Filter mode owns the keyboard: every printable key is search text, not a
	// command. Otherwise typing "d" to find "daemonset" would delete the cluster.
	if m.filtering {
		switch msg.Type {
		case tea.KeyRunes, tea.KeySpace:
			m.filter += string(msg.Runes)
			if msg.Type == tea.KeySpace {
				m.filter += " "
			}
			m.rebuildRows()
			m.mode = modeDetail
			m.refreshView()
			return m, nil
		case tea.KeyBackspace:
			if r := []rune(m.filter); len(r) > 0 {
				m.filter = string(r[:len(r)-1])
			}
			m.rebuildRows()
			m.mode = modeDetail
			m.refreshView()
			return m, nil
		case tea.KeyEnter:
			m.filtering = false // keep the filter, hand the keyboard back
			return m, nil
		case tea.KeyEsc, tea.KeyCtrlC:
			m.filtering, m.filter = false, ""
			m.rebuildRows()
			m.mode = modeDetail
			m.refreshView()
			return m, nil
		case tea.KeyUp, tea.KeyDown:
			// Move the cursor without leaving the filter — narrow, then pick.
			if msg.Type == tea.KeyUp && m.cursor > 0 {
				m.cursor--
			}
			if msg.Type == tea.KeyDown && m.cursor < len(m.sel)-1 {
				m.cursor++
			}
			m.clampListOff()
			m.refreshView()
			return m, nil
		}
		return m, nil
	}
	// Destroy-cluster guard: 'd' throws away every scenario in progress.
	if m.confirmDown {
		switch msg.String() {
		case "y", "Y":
			m.confirmDown = false
			return m, tea.ExecProcess(runner.Cmd(m.root, "down"), func(error) tea.Msg { return execDoneMsg{} })
		default:
			m.confirmDown = false
		}
		return m, nil
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
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Inits run up to 360s. Without this the only exit from a long or
			// wedged task is killing the whole TUI.
			if m.cancelRun != nil {
				m.cancelRun()
			}
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
		// One key backs out of whatever narrowed the view — first the filter,
		// then the pane mode.
		if m.filter != "" {
			m.filter = ""
			m.rebuildRows()
		}
		m.mode = modeDetail
		m.refreshView()
	case "/":
		m.filtering = true
		m.mode = modeDetail
		m.refreshView()
	case "n":
		if !m.nextUnsolved() {
			m.mode = modeOutput
			m.vp.SetContent(okStyle.Render("✓ every lesson in view is solved.") + "\n\n" +
				dimStyle.Render("Clear the filter with esc to search the whole course."))
			return m, nil
		}
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
		// Deleting the cluster discards every scenario in progress and costs
		// minutes to rebuild. Revealing a solution already asks for confirmation;
		// this is the more expensive of the two.
		m.confirmDown = true
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
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelRun = cancel
	root, name := m.root, l.Name
	return m, tea.Batch(
		m.spin.Tick,
		func() tea.Msg {
			out, ok := runner.CaptureContext(ctx, root, name, action)
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
	// That temp dir holds a cluster-admin kubeconfig. It is mode 0600, but it was
	// also never deleted — one copy leaked per shell opened, for the life of the
	// machine. Remove it when the shell exits.
	dir := filepath.Dir(rc)
	return tea.ExecProcess(c, func(error) tea.Msg {
		_ = os.RemoveAll(dir)
		return execDoneMsg{}
	})
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
