package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/madhank93/kubelings/internal/course"
	"github.com/madhank93/kubelings/internal/progress"
	"github.com/madhank93/kubelings/internal/runner"
)

func cloudLesson() *course.Lesson {
	return &course.Lesson{
		Name: "encryption-at-rest", Title: "Encryption at rest", Type: "lab",
		HasTasks: true, CloudOnly: true, CloudOnlyReason: "it reboots the node",
	}
}

// Cloud-only outranks the type badge: the column fits one, and "can't run here"
// is the more actionable fact.
func TestLessonBadgeCloudOutranksType(t *testing.T) {
	if got := lessonBadgeText(cloudLesson()); got != "☁cloud" {
		t.Errorf("cloud-only lab badge = %q, want ☁cloud", got)
	}
	plain := &course.Lesson{Name: "x", Type: "replay", HasTasks: true}
	if got := lessonBadgeText(plain); got != "⟲replay" {
		t.Errorf("non-cloud badge = %q, want ⟲replay", got)
	}
}

func TestBlockCloudOnly(t *testing.T) {
	m := &model{root: ".", vp: viewport.New(80, 24)}
	if m.blockCloudOnly(&course.Lesson{Name: "ok", HasTasks: true}) {
		t.Error("blocked a lesson that is not cloud-only")
	}
	if !m.blockCloudOnly(cloudLesson()) {
		t.Fatal("did not block a cloud-only lesson")
	}
	out := m.vp.View()
	for _, want := range []string{"iximiuz", "it reboots the node"} {
		if !strings.Contains(out, want) {
			t.Errorf("notice missing %q; got:\n%s", want, out)
		}
	}
}

// Every local-execution key must refuse a cloud-only lesson. `t` matters as much
// as the rest: the shell's rcfile wires `verify` straight to the local runner.
func TestCloudOnlyKeysNeverRunLocally(t *testing.T) {
	l := cloudLesson()
	for _, key := range []string{"i", "v", "r", "t"} {
		m := model{
			root: ".", vp: viewport.New(80, 24), mode: modeDetail,
			rows: []row{{lesson: l}}, sel: []int{0},
			prog:   map[string]progress.State{},
			status: runner.ClusterStatus{Up: true}, // cluster up: only the gate can stop it
		}
		got, cmd := m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		gm := got.(model)
		if gm.running {
			t.Errorf("key %q started a local run for a cloud-only lesson", key)
		}
		if cmd != nil {
			t.Errorf("key %q returned a command for a cloud-only lesson", key)
		}
		if !strings.Contains(gm.vp.View(), "iximiuz") {
			t.Errorf("key %q did not show the cloud-only notice; got:\n%s", key, gm.vp.View())
		}
	}
}

// ↵ self-attests instead of playing — the same affordance readings use.
func TestCloudOnlyEnterTogglesInsteadOfPlaying(t *testing.T) {
	m := model{
		root: t.TempDir(), vp: viewport.New(80, 24), mode: modeDetail,
		rows: []row{{lesson: cloudLesson()}}, sel: []int{0},
		prog:   map[string]progress.State{},
		status: runner.ClusterStatus{Up: true},
	}
	got, _ := m.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	gm := got.(model)
	if gm.running {
		t.Error("↵ started a local run for a cloud-only lesson")
	}
	if progress.Get(gm.prog, "encryption-at-rest") != progress.Solved {
		t.Error("↵ did not mark the cloud-only lesson done")
	}
}

// The bug this feature sat on top of: v/r on a reading marked it solved.
func TestReadingNeverRunsVerify(t *testing.T) {
	m := model{
		root: ".", vp: viewport.New(80, 24), mode: modeDetail,
		rows: []row{{lesson: &course.Lesson{Name: "cni-basics", Title: "CNI", Type: "read"}}},
		sel:  []int{0}, prog: map[string]progress.State{},
		status: runner.ClusterStatus{Up: true},
	}
	got, cmd := m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	if got.(model).running || cmd != nil {
		t.Error("v on a reading reached the runner")
	}
}
