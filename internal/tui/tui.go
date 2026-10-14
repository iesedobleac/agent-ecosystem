// Package tui is the interactive installer.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iesedobleac/agent-ecosystem/internal/installer"
	"github.com/iesedobleac/agent-ecosystem/internal/render"
)

var (
	accent   = lipgloss.Color("#7AA2F7")
	okColor  = lipgloss.Color("#9ECE6A")
	warnCol  = lipgloss.Color("#E0AF68")
	dim      = lipgloss.Color("#737AA2")
	title    = lipgloss.NewStyle().Bold(true).Foreground(accent)
	subtle   = lipgloss.NewStyle().Foreground(dim)
	selected = lipgloss.NewStyle().Foreground(accent).Bold(true)
	okStyle  = lipgloss.NewStyle().Foreground(okColor)
	warnSty  = lipgloss.NewStyle().Foreground(warnCol)
	box      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(dim).Padding(1, 2)
)

type screen int

const (
	scMenu screen = iota
	scTargets
	scComponents
	scModels
	scReview
	scChecking
	scRequirements
	scRunning
	scDone
)

type item struct {
	key, label, desc string
	checked          bool
}

type checklist struct {
	items  []item
	cursor int
}

func (c *checklist) update(key string) {
	switch key {
	case "up", "k":
		if c.cursor > 0 {
			c.cursor--
		}
	case "down", "j":
		if c.cursor < len(c.items)-1 {
			c.cursor++
		}
	case " ", "x":
		c.items[c.cursor].checked = !c.items[c.cursor].checked
	case "a":
		all := true
		for _, it := range c.items {
			all = all && it.checked
		}
		for i := range c.items {
			c.items[i].checked = !all
		}
	}
}

func (c checklist) view() string {
	var b strings.Builder
	for i, it := range c.items {
		mark := "[ ]"
		if it.checked {
			mark = "[x]"
		}
		var line string
		switch {
		case i == c.cursor:
			line = selected.Render("› " + mark + " " + it.label)
		case it.checked:
			line = "  " + okStyle.Render(mark) + " " + it.label
		default:
			line = "  " + mark + " " + it.label
		}
		b.WriteString(line)
		if it.desc != "" {
			b.WriteString(subtle.Render("  " + it.desc))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (c checklist) checked() []string {
	var out []string
	for _, it := range c.items {
		if it.checked {
			out = append(out, it.key)
		}
	}
	return out
}

type action int

const (
	actInstall action = iota
	actSync
	actDoctor
	actUninstall
	actQuit
)

var menuItems = []struct {
	act         action
	label, desc string
}{
	{actInstall, "Install or update", "choose assistants, components and models"},
	{actSync, "Sync", "re-apply the last selection (after an upgrade)"},
	{actDoctor, "Doctor", "read-only health check"},
	{actUninstall, "Uninstall", "remove everything this tool installed"},
	{actQuit, "Quit", ""},
}

// logLine is a log message, or with kind "step" an install step starting.
type logLine struct {
	kind, text  string
	done, total int
}

type reqsMsg []installer.Requirement

type doneMsg struct {
	res *installer.Result
	ok  bool
	err error
}

// modelsMsg carries the listed models per target; nil when they could not be listed.
type modelsMsg map[string][]string

// pickRows is how many models the picker shows at once.
const pickRows = 10

// Model is the Bubble Tea model.
type Model struct {
	screen     screen
	menu       int
	act        action
	detected   []string
	targets    checklist
	components checklist
	values     []string    // chosen model per modelKeys entry; empty until chosen
	modelKeys  [][2]string // target, profile
	modelFocus int         // row on the models screen; one past the last is Continue
	modelNote  string
	avail      map[string][]string // listed models per target; nil when they could not be listed
	listed     map[string]bool     // targets whose models were looked up
	picking    bool
	pickFilter textinput.Model
	pickCursor int
	pickCustom bool
	pickInput  textinput.Model
	review     int
	dryRun     bool
	pending    action            // what the requirements screen confirms
	pendingOpt installer.Options // options it runs with
	reqs       []installer.Requirement
	reqButton  int
	spin       spinner.Model
	bar        progress.Model
	step       logLine // last install step reported
	logs       []logLine
	logCh      chan logLine
	res        *installer.Result
	err        error
	ok         bool
	width      int
}

// New builds the model, preselecting the previous selection when there is one.
func New() Model {
	prev, _ := installer.ReadSelection("")
	detected := installer.DetectTargets(installer.Options{})
	m := Model{detected: detected}

	selectedTargets := detected
	comps := installer.DefaultComponents()
	if prev != nil {
		selectedTargets, comps = prev.Targets, prev.Components
	}
	for _, t := range installer.AllTargets {
		desc := ""
		if containsStr(detected, t) {
			desc = "detected"
		}
		m.targets.items = append(m.targets.items, item{key: t, label: installer.TargetLabels[t], desc: desc, checked: containsStr(selectedTargets, t)})
	}
	m.components.items = []item{
		{"sdd", "SDD flow", "sdd-orchestrator, spec/plan/tasks/apply/verify agents, sdd-workflow skill", comps.SDD},
		{"android", "Android orchestrator", "android-orchestrator (its explore/worker/review/device subagents come with either)", comps.Android},
		{"skills", "General skills", "skill-generator", comps.Skills},
		{"plugin", "Context warning", "OpenCode toast when a session passes 200k tokens", comps.Plugin},
		{"engram", "Engram memory", "persistent memory MCP and plugin", comps.Engram},
		{"codegraph", "CodeGraph", "code graph MCP", comps.CodeGraph},
	}

	models := map[string]render.Models{}
	if prev != nil && prev.Models != nil {
		models = prev.Models
	}
	// Defaults are filled in once the models are listed, and only if available.
	for _, t := range installer.AllTargets {
		for _, profile := range []string{"planner", "executor", "reviewer"} {
			m.values = append(m.values, models[t][profile])
			m.modelKeys = append(m.modelKeys, [2]string{t, profile})
		}
	}
	m.avail, m.listed = map[string][]string{}, map[string]bool{}
	m.pickFilter = textinput.New()
	m.pickFilter.Prompt = "Filter: "
	m.pickInput = textinput.New()
	m.pickInput.Prompt = "Model ID: "
	m.pickInput.CharLimit = 120
	m.pickInput.Width = 50
	m.spin = spinner.New()
	m.spin.Spinner = spinner.Dot
	m.bar = newBar()
	return m
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

func (m Model) options() installer.Options {
	opts := installer.Options{Targets: m.targets.checked(), DryRun: m.dryRun,
		Models: map[string]render.Models{}}
	comps := map[string]bool{}
	for _, k := range m.components.checked() {
		comps[k] = true
	}
	opts.Components = installer.Components{SDD: comps["sdd"], Android: comps["android"], Skills: comps["skills"],
		Plugin: comps["plugin"], Engram: comps["engram"], CodeGraph: comps["codegraph"]}
	for i, k := range m.modelKeys {
		if opts.Models[k[0]] == nil {
			opts.Models[k[0]] = render.Models{}
		}
		opts.Models[k[0]][k[1]] = m.values[i]
	}
	return opts
}

// modelsReady reports whether the models of every selected target were looked up.
func (m Model) modelsReady() bool {
	for _, t := range m.targets.checked() {
		if !m.listed[t] {
			return false
		}
	}
	return true
}

// enterModels opens the models screen, listing the models it has not looked up yet.
func (m Model) enterModels() (Model, tea.Cmd) {
	m.screen, m.modelFocus, m.modelNote, m.picking = scModels, 0, "", false
	var todo []string
	for _, t := range m.targets.checked() {
		if !m.listed[t] {
			todo = append(todo, t)
		}
	}
	if len(todo) == 0 {
		return m, nil
	}
	return m, tea.Batch(m.spin.Tick, func() tea.Msg {
		out := modelsMsg{}
		for _, t := range todo {
			if list, ok := installer.AvailableModels(t); ok {
				out[t] = list
			} else {
				out[t] = nil
			}
		}
		return out
	})
}

// pickOptions are the listed models of the target being picked that match the filter.
func (m Model) pickOptions() []string {
	t := m.modelKeys[m.visibleModels()[m.modelFocus]][0]
	filter := strings.ToLower(strings.TrimSpace(m.pickFilter.Value()))
	var out []string
	for _, v := range m.avail[t] {
		if strings.Contains(strings.ToLower(v), filter) {
			out = append(out, v)
		}
	}
	return out
}

// openPicker starts choosing the model of the focused row.
func (m Model) openPicker() (Model, tea.Cmd) {
	i := m.visibleModels()[m.modelFocus]
	m.picking, m.pickCursor, m.modelNote = true, 0, ""
	m.pickFilter.SetValue("")
	m.pickInput.SetValue(m.values[i])
	m.pickCustom = m.avail[m.modelKeys[i][0]] == nil
	if m.pickCustom {
		m.pickFilter.Blur()
		return m, m.pickInput.Focus()
	}
	for n, v := range m.pickOptions() {
		if v == m.values[i] {
			m.pickCursor = n
		}
	}
	m.pickInput.Blur()
	return m, m.pickFilter.Focus()
}

func (m Model) handlePicker(msg tea.KeyMsg, key string) (Model, tea.Cmd) {
	i := m.visibleModels()[m.modelFocus]
	var cmd tea.Cmd
	if m.pickCustom {
		switch key {
		case "enter":
			if v := strings.TrimSpace(m.pickInput.Value()); v != "" {
				m.values[i], m.picking = v, false
			}
		case "esc":
			if m.avail[m.modelKeys[i][0]] == nil {
				m.picking = false
			} else {
				m.pickCustom = false
				m.pickInput.Blur()
				cmd = m.pickFilter.Focus()
			}
		default:
			m.pickInput, cmd = m.pickInput.Update(msg)
		}
		return m, cmd
	}
	opts := m.pickOptions()
	switch key {
	case "up":
		if m.pickCursor > 0 {
			m.pickCursor--
		}
	case "down":
		if m.pickCursor < len(opts) { // the last row is Custom…
			m.pickCursor++
		}
	case "enter":
		if m.pickCursor < len(opts) {
			m.values[i], m.picking = opts[m.pickCursor], false
			return m, nil
		}
		m.pickCustom = true
		if f := strings.TrimSpace(m.pickFilter.Value()); f != "" {
			m.pickInput.SetValue(f)
		}
		m.pickFilter.Blur()
		return m, m.pickInput.Focus()
	case "esc":
		m.picking = false
	default:
		m.pickFilter, cmd = m.pickFilter.Update(msg)
		m.pickCursor = 0
	}
	return m, cmd
}

func (m Model) pickerView() string {
	var b strings.Builder
	i := m.visibleModels()[m.modelFocus]
	k := m.modelKeys[i]
	b.WriteString(title.Render(fmt.Sprintf("%s %s model", installer.TargetLabels[k[0]], k[1])) + "\n\n")
	if m.pickCustom {
		if m.avail[k[0]] == nil {
			b.WriteString(warnSty.Render("Could not list the "+installer.TargetLabels[k[0]]+" models: connect a provider there, or type a model ID.") + "\n\n")
		}
		b.WriteString(m.pickInput.View() + "\n")
		return b.String() + help("type a model ID · enter confirm · esc back")
	}
	b.WriteString(m.pickFilter.View() + "\n\n")
	opts := append(m.pickOptions(), "Custom…")
	start := 0
	if m.pickCursor >= pickRows {
		start = m.pickCursor - pickRows + 1
	}
	for n := start; n < len(opts) && n < start+pickRows; n++ {
		label := opts[n]
		switch {
		case n == m.pickCursor:
			b.WriteString(selected.Render("› "+label) + "\n")
		case label == m.values[i]:
			b.WriteString("  " + okStyle.Render(label) + "\n")
		default:
			b.WriteString("  " + label + "\n")
		}
	}
	if len(opts) > pickRows {
		b.WriteString(subtle.Render(fmt.Sprintf("  %d models, type to filter", len(opts)-1)) + "\n")
	}
	return b.String() + help("↑/↓ move · type to filter · enter choose · esc back")
}

// visibleModels are the model inputs for the selected targets.
func (m Model) visibleModels() []int {
	var idx []int
	sel := m.targets.checked()
	for i, k := range m.modelKeys {
		if containsStr(sel, k[0]) {
			idx = append(idx, i)
		}
	}
	return idx
}

type chanLogger chan logLine

func (c chanLogger) Info(s string) { c <- logLine{kind: "info", text: s} }
func (c chanLogger) OK(s string)   { c <- logLine{kind: "ok", text: s} }
func (c chanLogger) Warn(s string) { c <- logLine{kind: "warn", text: s} }

// Step implements installer.Progress.
func (c chanLogger) Step(done, total int, label string) {
	c <- logLine{kind: "step", text: label, done: done, total: total}
}

func newBar() progress.Model {
	return progress.New(progress.WithGradient(string(accent), string(okColor)), progress.WithWidth(40))
}

func waitLog(ch chan logLine) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return nil
		}
		return line
	}
}

func (m Model) start(act action, opts installer.Options) (Model, tea.Cmd) {
	m.screen, m.act, m.logs, m.step, m.bar = scRunning, act, nil, logLine{}, newBar()
	m.logCh = make(chan logLine, 256)
	ch := m.logCh
	run := func() tea.Msg {
		log := chanLogger(ch)
		var msg doneMsg
		switch act {
		case actInstall:
			msg.res, msg.err = installer.Install(opts, log)
			msg.ok = msg.err == nil
		case actSync:
			prev, err := installer.ReadSelection("")
			if err != nil {
				msg.err = err
				break
			}
			msg.res, msg.err = installer.Install(*prev, log)
			msg.ok = msg.err == nil
		case actDoctor:
			msg.ok = installer.Doctor(installer.Options{}, log)
		case actUninstall:
			msg.err = installer.Uninstall(installer.Options{}, log)
			msg.ok = msg.err == nil
		}
		close(ch)
		return msg
	}
	return m, tea.Batch(m.spin.Tick, run, waitLog(ch))
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.bar.Width = max(20, min(60, msg.Width-16))
		return m, nil
	case logLine:
		if msg.kind == "step" {
			m.step = msg
			return m, tea.Batch(m.bar.SetPercent(float64(msg.done)/float64(max(msg.total, 1))), waitLog(m.logCh))
		}
		m.logs = append(m.logs, msg)
		return m, waitLog(m.logCh)
	case progress.FrameMsg:
		bar, cmd := m.bar.Update(msg)
		m.bar = bar.(progress.Model)
		return m, cmd
	case reqsMsg:
		m.screen, m.reqs, m.reqButton = scRequirements, msg, 0
		if m.blocked() {
			m.reqButton = 1
		}
		return m, nil
	case doneMsg:
		m.screen, m.res, m.err, m.ok = scDone, msg.res, msg.err, msg.ok
		return m, nil
	case modelsMsg:
		for t, list := range msg {
			m.avail[t], m.listed[t] = list, true
		}
		// Preselect a default only where the assistant lists it; otherwise the user chooses.
		for i, k := range m.modelKeys {
			if _, ok := msg[k[0]]; ok && m.values[i] == "" {
				if def := installer.DefaultModels[k[0]][k[1]]; installer.KnownModel(k[0], def, m.avail[k[0]]) {
					m.values[i] = def
				}
			}
		}
		return m, nil
	case spinner.TickMsg:
		if m.screen == scRunning || m.screen == scChecking || (m.screen == scModels && !m.modelsReady()) {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleKey(msg, key)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	switch m.screen {
	case scMenu:
		switch key {
		case "up", "k":
			if m.menu > 0 {
				m.menu--
			}
		case "down", "j":
			if m.menu < len(menuItems)-1 {
				m.menu++
			}
		case "q", "esc":
			return m, tea.Quit
		case "enter":
			switch menuItems[m.menu].act {
			case actInstall:
				m.screen = scTargets
			case actSync:
				prev, err := installer.ReadSelection("")
				if err != nil {
					m.screen, m.err, m.ok, m.res, m.logs = scDone, err, false, nil, nil
					return m, nil
				}
				return m.check(actSync, *prev)
			case actDoctor:
				return m.start(actDoctor, installer.Options{})
			case actUninstall:
				return m.start(actUninstall, installer.Options{})
			case actQuit:
				return m, tea.Quit
			}
		}
	case scTargets, scComponents:
		list := &m.targets
		next, prev := scComponents, scMenu
		if m.screen == scComponents {
			list, next, prev = &m.components, scModels, scTargets
		}
		switch key {
		case "enter":
			if m.screen == scTargets && len(m.targets.checked()) == 0 {
				return m, nil
			}
			if next == scModels {
				return m.enterModels()
			}
			m.screen = next
		case "esc":
			m.screen = prev
		default:
			list.update(key)
		}
	case scModels:
		if m.picking {
			return m.handlePicker(msg, key)
		}
		vis := m.visibleModels()
		switch key {
		case "esc":
			m.screen = scComponents
		case "up", "k", "shift+tab":
			if m.modelFocus > 0 {
				m.modelFocus--
			}
		case "down", "j", "tab":
			if m.modelFocus < len(vis) {
				m.modelFocus++
			}
		case "enter", " ":
			if !m.modelsReady() {
				return m, nil
			}
			if m.modelFocus < len(vis) {
				return m.openPicker()
			}
			for _, i := range vis {
				if m.values[i] == "" {
					m.modelNote = "Choose a model for every row."
					return m, nil
				}
			}
			m.screen, m.review = scReview, 0
		}
	case scReview:
		switch key {
		case "left", "h", "up", "k":
			if m.review > 0 {
				m.review--
			}
		case "right", "l", "down", "j", "tab":
			if m.review < 2 {
				m.review++
			}
		case "esc":
			m.screen, m.modelFocus, m.modelNote = scModels, 0, ""
		case "enter":
			switch m.review {
			case 0:
				m.dryRun = false
				return m.check(actInstall, m.options())
			case 1:
				m.dryRun = true
				return m.check(actInstall, m.options())
			case 2:
				m.screen = scMenu
			}
		}
	case scRequirements:
		switch key {
		case "left", "h", "up", "k":
			if m.reqButton > 0 && !m.blocked() {
				m.reqButton--
			}
		case "right", "l", "down", "j", "tab":
			if m.reqButton < 1 {
				m.reqButton++
			}
		case "esc":
			m.screen = m.back()
		case "enter":
			if m.reqButton == 0 {
				return m.start(m.pending, m.pendingOpt)
			}
			m.screen = m.back()
		}
	case scDone:
		switch key {
		case "enter", "esc":
			m.screen = scMenu
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func help(keys string) string { return "\n" + subtle.Render(keys) }

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder
	b.WriteString(header(m.width) + "\n\n")
	switch m.screen {
	case scMenu:
		for i, it := range menuItems {
			if i == m.menu {
				b.WriteString(selected.Render("› "+it.label) + subtle.Render("  "+it.desc) + "\n")
			} else {
				b.WriteString("  " + it.label + "\n")
			}
		}
		b.WriteString(help("↑/↓ move · enter select · q quit"))
	case scTargets:
		b.WriteString(title.Render("Assistants") + subtle.Render("  (1/4)") + "\n\n" + m.targets.view())
		b.WriteString(help("space toggle · a all · enter next · esc back"))
	case scComponents:
		b.WriteString(title.Render("Components") + subtle.Render("  (2/4)") + "\n\n" + m.components.view())
		b.WriteString(help("space toggle · a all · enter next · esc back"))
	case scModels:
		b.WriteString(title.Render("Models") + subtle.Render("  (3/4) planner: orchestrators and planning · executor: worker, device, apply · reviewer: review, verify") + "\n\n")
		switch {
		case !m.modelsReady():
			b.WriteString(m.spin.View() + " Listing the available models…\n")
		case m.picking:
			b.WriteString(m.pickerView())
		default:
			vis := m.visibleModels()
			for n, i := range vis {
				k := m.modelKeys[i]
				label := fmt.Sprintf("%-12s %-9s", installer.TargetLabels[k[0]], k[1])
				value := m.values[i]
				if value == "" {
					value = warnSty.Render("choose a model")
				}
				if n == m.modelFocus {
					b.WriteString(selected.Render("› "+label) + " " + value + "\n")
				} else {
					b.WriteString("  " + label + " " + value + "\n")
				}
			}
			cont := "Continue →"
			if m.modelFocus == len(vis) {
				cont = selected.Render("› " + cont)
			} else {
				cont = "  " + cont
			}
			b.WriteString("\n" + cont + "\n")
			if m.modelNote != "" {
				b.WriteString("\n" + warnSty.Render(m.modelNote) + "\n")
			}
			b.WriteString(help("↑/↓ move · enter choose · esc back"))
		}
	case scReview:
		b.WriteString(title.Render("Review") + subtle.Render("  (4/4)") + "\n\n" + box.Render(m.reviewView()) + "\n\n")
		buttons := []string{"Install", "Dry run", "Cancel"}
		for i, btn := range buttons {
			if i == m.review {
				b.WriteString(selected.Render("[ "+btn+" ]") + "  ")
			} else {
				b.WriteString("  " + btn + "    ")
			}
		}
		b.WriteString("\n" + help("←/→ choose · enter confirm · esc back"))
	case scChecking:
		b.WriteString(m.spin.View() + " Checking the tools this selection needs…\n")
	case scRequirements:
		b.WriteString(title.Render("Required tools") + subtle.Render("  checked before anything is written") + "\n\n")
		b.WriteString(requirementsView(m.reqs) + "\n")
		blocked := m.blocked()
		switch {
		case blocked:
			b.WriteString(warnSty.Render("Install the tools marked ✗ and run again.") + "\n\n")
		case m.dryRun || m.pendingOpt.DryRun:
			b.WriteString(subtle.Render("Dry run: nothing is written.") + "\n\n")
		}
		label := "Install"
		if m.pending == actSync {
			label = "Sync"
		}
		if m.pendingOpt.DryRun {
			label = "Dry run"
		}
		buttons := []string{label, "Back"}
		for i, btn := range buttons {
			switch {
			case i == 0 && blocked:
				b.WriteString(subtle.Render("  " + btn + "    "))
			case i == m.reqButton:
				b.WriteString(selected.Render("[ "+btn+" ]") + "  ")
			default:
				b.WriteString("  " + btn + "    ")
			}
		}
		b.WriteString("\n" + help("←/→ choose · enter confirm · esc back"))
	case scRunning:
		if m.step.total == 0 {
			b.WriteString(m.spin.View() + " Working…\n\n" + m.logView(14))
			break
		}
		b.WriteString(title.Render("Installing") + "\n\n")
		b.WriteString(m.bar.View() + subtle.Render(fmt.Sprintf("  %d/%d", m.step.done, m.step.total)) + "\n")
		b.WriteString(m.spin.View() + " " + m.step.text + "\n\n" + m.logView(12))
	case scDone:
		switch {
		case m.err != nil:
			b.WriteString(warnSty.Render("✗ "+m.err.Error()) + "\n\n")
		case m.ok:
			b.WriteString(okStyle.Render("✓ Done") + "\n\n")
		default:
			b.WriteString(warnSty.Render("! Finished with warnings") + "\n\n")
		}
		b.WriteString(m.logView(18))
		if m.res != nil && len(m.res.NextSteps) > 0 {
			b.WriteString("\n" + title.Render("Next steps") + "\n")
			for i, s := range m.res.NextSteps {
				b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, s))
			}
		}
		if m.res != nil && m.res.BackupDir != "" {
			b.WriteString(subtle.Render("\nBackups: "+m.res.BackupDir) + "\n")
		}
		b.WriteString(help("enter menu · q quit"))
	}
	return b.String() + "\n"
}

// check computes the requirements for a pending install or sync.
func (m Model) check(act action, opts installer.Options) (Model, tea.Cmd) {
	if len(opts.Targets) == 0 {
		opts.Targets = installer.DetectTargets(opts)
	}
	m.screen, m.pending, m.pendingOpt = scChecking, act, opts
	return m, tea.Batch(m.spin.Tick, func() tea.Msg { return reqsMsg(installer.Requirements(opts)) })
}

func (m Model) blocked() bool {
	return !m.pendingOpt.DryRun && len(installer.Blockers(m.reqs)) > 0
}

// back returns to the screen before the requirements check.
func (m Model) back() screen {
	if m.pending == actSync {
		return scMenu
	}
	return scReview
}

func requirementsView(reqs []installer.Requirement) string {
	var b strings.Builder
	width := 0
	for _, r := range reqs {
		width = max(width, len(r.Name))
	}
	for _, r := range reqs {
		name := fmt.Sprintf("%-*s", width, r.Name)
		switch r.Status {
		case installer.Present:
			b.WriteString(okStyle.Render("✓ ") + name + "  " + subtle.Render(r.Detail) + "\n")
		case installer.WillInstall:
			b.WriteString(selected.Render("↓ ") + name + "  will be installed " + r.Detail + "\n")
		case installer.Optional:
			b.WriteString(warnSty.Render("! ") + name + "  missing: " + r.Hint + "\n")
		case installer.Blocking:
			b.WriteString(warnSty.Bold(true).Render("✗ "+name) + "  required: " + r.Hint + "\n")
		}
		b.WriteString(subtle.Render(fmt.Sprintf("  %-*s  %s", width, "", r.Purpose)) + "\n")
	}
	return b.String()
}

func (m Model) logView(max int) string {
	logs := m.logs
	if len(logs) > max {
		logs = logs[len(logs)-max:]
	}
	var b strings.Builder
	for _, l := range logs {
		switch l.kind {
		case "ok":
			b.WriteString(okStyle.Render("✓ ") + l.text + "\n")
		case "warn":
			b.WriteString(warnSty.Render("! ") + l.text + "\n")
		default:
			b.WriteString(subtle.Render("• ") + l.text + "\n")
		}
	}
	return b.String()
}

// reviewView summarizes the selection in sections: assistants, components and models.
func (m Model) reviewView() string {
	opts := m.options()
	section := lipgloss.NewStyle().Bold(true)
	var b strings.Builder

	b.WriteString(section.Render("Assistants") + "\n")
	for _, t := range opts.Targets {
		b.WriteString("  " + okStyle.Render("✓") + " " + installer.TargetLabels[t] + "\n")
	}

	// Components in two columns: chosen ones ticked, the rest dimmed.
	b.WriteString("\n" + section.Render("Components") + "\n")
	width := 0
	for _, it := range m.components.items {
		width = max(width, len(it.label))
	}
	half := (len(m.components.items) + 1) / 2
	for r := 0; r < half; r++ {
		line := "  " + componentCell(m.components.items[r], width)
		if r+half < len(m.components.items) {
			line += "    " + componentCell(m.components.items[r+half], width)
		}
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}

	// Models as a table: one row per assistant, one column per profile.
	profiles := []string{"planner", "executor", "reviewer"}
	widths := map[string]int{}
	labelW := 0
	for _, t := range opts.Targets {
		labelW = max(labelW, len(installer.TargetLabels[t]))
		for _, p := range profiles {
			widths[p] = max(widths[p], len(p), len(opts.Models[t][p]))
		}
	}
	b.WriteString("\n" + section.Render("Models") + "\n")
	head := fmt.Sprintf("  %-*s", labelW, "")
	for _, p := range profiles {
		head += fmt.Sprintf("  %-*s", widths[p], p)
	}
	b.WriteString(subtle.Render(strings.TrimRight(head, " ")) + "\n")
	for _, t := range opts.Targets {
		row := fmt.Sprintf("  %-*s", labelW, installer.TargetLabels[t])
		for _, p := range profiles {
			row += fmt.Sprintf("  %-*s", widths[p], opts.Models[t][p])
		}
		b.WriteString(strings.TrimRight(row, " ") + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func componentCell(it item, width int) string {
	if it.checked {
		return okStyle.Render("✓") + " " + fmt.Sprintf("%-*s", width, it.label)
	}
	return subtle.Render("– " + fmt.Sprintf("%-*s", width, it.label))
}

// Run starts the interactive installer.
func Run() error {
	_, err := tea.NewProgram(New(), tea.WithAltScreen()).Run()
	return err
}
