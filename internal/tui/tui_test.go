package tui

import (
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

	next, _ := m.Update(modelsMsg{"opencode": {"openai/gpt-5.5", "openai/gpt-5.6-terra"}, "claude": installer.ClaudeModels})
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
	next, _ := m.Update(modelsMsg{"opencode": nil})
	m = next.(Model)
	m = press(t, m, enter)
	if !m.picking || !m.pickCustom {
		t.Fatal("expected the model ID input when nothing is listed")
	}
	m = press(t, m, typed("p/m"), enter)
	if v := m.options().Models["opencode"]["planner"]; v != "p/m" {
		t.Errorf("planner %q", v)
	}
}
