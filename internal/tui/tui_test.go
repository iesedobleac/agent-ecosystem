package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iesedobleac/agent-ecosystem/internal/installer"
)

func press(t *testing.T, m Model, keys ...tea.KeyMsg) Model {
	t.Helper()
	for _, k := range keys {
		next, _ := m.Update(k)
		m = next.(Model)
	}
	return m
}

var (
	enter = tea.KeyMsg{Type: tea.KeyEnter}
	down  = tea.KeyMsg{Type: tea.KeyDown}
)

func typed(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestModelsScreen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m := New()
	for i := range m.targets.items {
		m.targets.items[i].checked = true
	}
	m.screen = scComponents
	m = press(t, m, enter)
	if m.screen != scModels || m.modelsReady() {
		t.Fatalf("screen %d, ready %v", m.screen, m.modelsReady())
	}
	m = press(t, m, enter) // ignored while listing
	if m.picking {
		t.Fatal("picker opened before the models were listed")
	}

	next, _ := m.Update(modelsMsg{lists: map[string][]string{"opencode": {"openai/gpt-5.5", "openai/gpt-5.6-terra"}, "claude": installer.ClaudeModels}})
	m = next.(Model)
	opts := m.options()
	if opts.Models["claude"]["planner"] != "opus" || opts.Models["claude"]["reviewer"] != "sonnet" {
		t.Errorf("Claude defaults not preselected: %v", opts.Models["claude"])
	}
	if v := opts.Models["opencode"]["planner"]; v != "" {
		t.Errorf("unlisted OpenCode default preselected: %q", v)
	}

	// Continue is refused while a row has no model.
	m = press(t, m, down, down, down, down, down, down, enter)
	if m.screen != scModels || m.modelNote == "" {
		t.Fatalf("continued without models: screen %d", m.screen)
	}

	// Planner: filter and choose.
	m.modelFocus = 0
	m = press(t, m, enter, typed("terra"))
	if got := m.pickOptions(); len(got) != 1 {
		t.Fatalf("filtered options: %v", got)
	}
	m = press(t, m, enter)
	// Executor: Custom… with a typed ID.
	m = press(t, m, down, enter, down, down, enter, typed("other/model"), enter)
	// Reviewer: first listed model.
	m = press(t, m, down, enter, enter)
	if m.picking {
		t.Fatal("picker still open")
	}
	got := m.options().Models["opencode"]
	if got["planner"] != "openai/gpt-5.6-terra" || got["executor"] != "other/model" || got["reviewer"] != "openai/gpt-5.5" {
		t.Errorf("OpenCode models: %v", got)
	}

	m.modelFocus = len(m.visibleModels())
	m = press(t, m, enter)
	if m.screen != scReview {
		t.Errorf("screen %d after Continue", m.screen)
	}
}

func TestPickerWithoutList(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m := New()
	for i := range m.targets.items {
		m.targets.items[i].checked = m.targets.items[i].key == "opencode"
	}
	m.screen = scComponents
	m = press(t, m, enter)
	next, _ := m.Update(modelsMsg{lists: map[string][]string{"opencode": nil}, errs: map[string]string{"opencode": "opencode models: timed out after 60s"}})
	m = next.(Model)
	m = press(t, m, enter)
	if !m.picking || !m.pickCustom {
		t.Fatal("expected the model ID input when nothing is listed")
	}
	if view := m.View(); !strings.Contains(view, "timed out after 60s") {
		t.Errorf("reason not shown:\n%s", view)
	}
	m = press(t, m, typed("p/m"), enter)
	if v := m.options().Models["opencode"]["planner"]; v != "p/m" {
		t.Errorf("planner %q", v)
	}
}

func TestReviewView(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m := New()
	for i := range m.targets.items {
		m.targets.items[i].checked = true
	}
	for i := range m.components.items {
		m.components.items[i].checked = m.components.items[i].key != "codegraph"
	}
	for i, k := range m.modelKeys {
		m.values[i] = installer.DefaultModels[k[0]][k[1]]
	}
	view := m.reviewView()
	for _, want := range []string{"Assistants", "Components", "Models", "– CodeGraph", "planner", "github-copilot/claude-opus-5.5"} {
		if !strings.Contains(view, want) {
			t.Errorf("review view missing %q:\n%s", want, view)
		}
	}
}

func TestProgressSteps(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m := New()
	m.screen, m.logCh = scRunning, make(chan logLine, 1)
	next, _ := m.Update(logLine{kind: "step", text: "OpenCode: rules", done: 3, total: 14})
	m = next.(Model)
	if m.step.done != 3 || len(m.logs) != 0 {
		t.Fatalf("step %+v, logs %v", m.step, m.logs)
	}
	if view := m.View(); !strings.Contains(view, "3/14") || !strings.Contains(view, "OpenCode: rules") {
		t.Errorf("running view:\n%s", view)
	}
}

func TestHeaderFitsWidth(t *testing.T) {
	if h := header(100); !strings.Contains(h, "███") || !strings.Contains(h, "E C O S Y S T E M") {
		t.Errorf("wide header:\n%s", h)
	}
	if h := header(40); strings.Contains(h, "███") || !strings.Contains(h, "agent-ecosystem") {
		t.Errorf("narrow header:\n%s", h)
	}
}
