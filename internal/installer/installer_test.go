package installer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iesedobleac/agent-ecosystem/internal/render"
)

type testLogger struct{ t *testing.T }

func (l testLogger) Info(s string) { l.t.Log("info: " + s) }
func (l testLogger) OK(s string)   { l.t.Log("ok: " + s) }
func (l testLogger) Warn(s string) { l.t.Log("warn: " + s) }

func init() {
	// CI machines have no assistants installed; requirements are tested separately.
	checkRequirements = func(Options) []Requirement { return nil }
	listModels = func(string) ([]string, error) { return nil, errors.New("not listed in tests") }
}

type warnLogger struct{ warns *[]string }

func (warnLogger) Info(string)     {}
func (warnLogger) OK(string)       {}
func (l warnLogger) Warn(s string) { *l.warns = append(*l.warns, s) }

type stepLogger struct {
	nopLogger
	steps *[][2]int
}

func (l stepLogger) Step(done, total int, _ string) { *l.steps = append(*l.steps, [2]int{done, total}) }

func TestTargetRegistry(t *testing.T) {
	if strings.Join(AllTargets, ",") != "opencode,claude" {
		t.Errorf("AllTargets: %v", AllTargets)
	}
	for _, tg := range targets {
		id := tg.id()
		if targetByID(id) == nil || TargetLabels[id] == "" || tg.binary() == "" || tg.steps() <= 0 {
			t.Errorf("%s: incomplete adapter", id)
		}
		for _, p := range []string{"planner", "executor", "reviewer"} {
			if DefaultModels[id][p] == "" {
				t.Errorf("%s: no default %s model", id, p)
			}
		}
	}
	if targetByID("nope") != nil {
		t.Error("unknown target resolved")
	}
}

func TestProgress(t *testing.T) {
	var steps [][2]int
	if _, err := Install(testOptions(t.TempDir()), stepLogger{steps: &steps}); err != nil {
		t.Fatal(err)
	}
	total := 5 + openCode{}.steps() + claudeCode{}.steps()
	if len(steps) != total+1 {
		t.Fatalf("%d steps reported, want %d: %v", len(steps), total+1, steps)
	}
	for i, s := range steps {
		if s != [2]int{i, total} {
			t.Errorf("step %d reported as %v", i, s)
		}
	}
}

func TestCheckModels(t *testing.T) {
	saved := listModels
	defer func() { listModels = saved }()
	listModels = func(target string) ([]string, error) {
		if target == "opencode" {
			return []string{"openai/gpt-5.5"}, nil
		}
		return ClaudeModels, nil
	}
	opts := testOptions(t.TempDir())
	opts.Models = map[string]render.Models{
		"opencode": {"planner": "openai/gpt-5.5", "executor": "openai/gpt-5.5", "reviewer": "openai/missing"},
		"claude":   {"planner": "opus", "executor": "claude-haiku-4-5-20251001", "reviewer": "gpt"},
	}
	var warns []string
	if _, err := Install(opts, warnLogger{&warns}); err != nil {
		t.Fatal(err)
	}
	var models []string
	for _, w := range warns {
		if strings.Contains(w, "not among the available models") {
			models = append(models, w)
		}
	}
	if len(models) != 2 || !strings.Contains(models[0], `"openai/missing"`) || !strings.Contains(models[1], `"gpt"`) {
		t.Errorf("model warnings: %q", models)
	}
}

func testOptions(home string) Options {
	return Options{
		Targets:        []string{"opencode", "claude"},
		Components:     DefaultComponents(),
		ConfigDir:      filepath.Join(home, ".config", "opencode"),
		ClaudeDir:      filepath.Join(home, ".claude-test"),
		SkipDeps:       true,
		NoShellProfile: true,
		Home:           home,
	}
}

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			n++
		}
		return nil
	})
	return n
}

func TestInstallReinstallUninstall(t *testing.T) {
	home := t.TempDir()
	opts := testOptions(home)
	ocDir, clDir := opts.ConfigDir, opts.ClaudeDir

	// Existing user configuration that must survive.
	if err := os.MkdirAll(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(ocDir, "AGENTS.md"), []byte("# Mine\n"), 0o644)
	os.WriteFile(filepath.Join(ocDir, "opencode.jsonc"), []byte(`{
  // comment
  "share": "disabled",
  "mcp": {"servers": {"mine": {"type": "local", "command": ["x"]}}},
}`), 0o644)

	res, err := Install(opts, testLogger{t})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.NextSteps) == 0 {
		t.Error("no next steps")
	}
	ocAgents := countFiles(t, filepath.Join(ocDir, "agents"))
	clAgents := countFiles(t, filepath.Join(clDir, "agents"))
	if ocAgents != 14 || clAgents != 14 {
		t.Fatalf("agents: opencode %d, claude %d", ocAgents, clAgents)
	}
	agentsMd, _ := os.ReadFile(filepath.Join(ocDir, "AGENTS.md"))
	if !strings.HasPrefix(string(agentsMd), "# Mine\n") || strings.Count(string(agentsMd), markStart) != 1 {
		t.Errorf("AGENTS.md:\n%s", agentsMd)
	}
	config, err := readJSON(filepath.Join(ocDir, "opencode.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	servers := config["mcp"].(map[string]any)["servers"].(map[string]any)
	for _, s := range []string{"mine", "engram", "codegraph"} {
		if _, ok := servers[s]; !ok {
			t.Errorf("OpenCode MCP %s missing", s)
		}
	}
	claudeJSON, err := readJSON(filepath.Join(clDir, ".claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(claudeJSON["mcpServers"].(map[string]any)) != 2 {
		t.Errorf("Claude MCP servers: %v", claudeJSON["mcpServers"])
	}
	settings, _ := readJSON(filepath.Join(clDir, "settings.json"))
	if len(settings["permissions"].(map[string]any)["deny"].([]any)) != len(claudeDeny) {
		t.Errorf("deny rules: %v", settings)
	}

	// Re-running changes nothing.
	before, _ := os.ReadFile(filepath.Join(ocDir, "AGENTS.md"))
	if _, err := Install(opts, testLogger{t}); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(ocDir, "AGENTS.md"))
	if string(before) != string(after) {
		t.Error("AGENTS.md changed on re-run")
	}
	settings, _ = readJSON(filepath.Join(clDir, "settings.json"))
	if len(settings["permissions"].(map[string]any)["deny"].([]any)) != len(claudeDeny) {
		t.Error("deny rules duplicated on re-run")
	}

	// A hand-edited file is backed up before it is replaced.
	edited := filepath.Join(ocDir, "agents", "sdd-apply.md")
	os.WriteFile(edited, []byte("edited"), 0o644)
	res, err = Install(opts, testLogger{t})
	if err != nil {
		t.Fatal(err)
	}
	if res.BackupDir == "" {
		t.Error("no backup of the edited file")
	}

	// Sync reuses the saved selection.
	saved, err := ReadSelection(home)
	if err != nil || len(saved.Targets) != 2 {
		t.Fatalf("saved selection: %v %v", saved, err)
	}

	if err := Uninstall(Options{Home: home}, testLogger{t}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ocDir, "agents")); !os.IsNotExist(err) {
		t.Error("OpenCode agents dir left behind")
	}
	agentsMd, _ = os.ReadFile(filepath.Join(ocDir, "AGENTS.md"))
	if string(agentsMd) != "# Mine\n" {
		t.Errorf("AGENTS.md after uninstall: %q", agentsMd)
	}
	config, _ = readJSON(filepath.Join(ocDir, "opencode.jsonc"))
	servers = config["mcp"].(map[string]any)["servers"].(map[string]any)
	if len(servers) != 1 || servers["mine"] == nil {
		t.Errorf("OpenCode MCP after uninstall: %v", servers)
	}
	if _, err := os.Stat(filepath.Join(clDir, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Error("CLAUDE.md left behind")
	}
	var state State
	data, _ := os.ReadFile(filepath.Join(home, ".agent-ecosystem", "state.json"))
	_ = json.Unmarshal(data, &state)
	if len(state.Targets) != 0 {
		t.Errorf("state after uninstall: %v", state.Targets)
	}
}

func TestComponents(t *testing.T) {
	home := t.TempDir()
	opts := testOptions(home)
	opts.Targets = []string{"claude"}
	opts.Components.Android = false
	if _, err := Install(opts, testLogger{t}); err != nil {
		t.Fatal(err)
	}
	dir := opts.ClaudeDir
	if _, err := os.Stat(filepath.Join(dir, "agents", "android-orchestrator.md")); !os.IsNotExist(err) {
		t.Error("android-orchestrator installed without the Android component")
	}
	if _, err := os.Stat(filepath.Join(dir, "agents", "android-worker.md")); err != nil {
		t.Error("android-worker should come with SDD")
	}
	claudeJSON, _ := readJSON(filepath.Join(dir, ".claude.json"))
	if claudeJSON["mcpServers"].(map[string]any)["codegraph"] == nil {
		t.Error("codegraph MCP missing")
	}

	// Dropping a component removes what it added.
	opts.Components.CodeGraph = false
	opts.Components.SDD = false
	if _, err := Install(opts, testLogger{t}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "skills", "sdd-workflow")); !os.IsNotExist(err) {
		t.Error("sdd-workflow skill left behind")
	}
	if _, err := os.Stat(filepath.Join(dir, "agents", "android-worker.md")); !os.IsNotExist(err) {
		t.Error("android-worker left behind without SDD or Android")
	}
	claudeJSON, _ = readJSON(filepath.Join(dir, ".claude.json"))
	if claudeJSON["mcpServers"].(map[string]any)["codegraph"] != nil {
		t.Error("codegraph MCP left behind")
	}
}

func TestRequirements(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // nothing is installed
	opts := Options{Targets: []string{"opencode", "claude"}, Components: DefaultComponents()}
	byName := map[string]Requirement{}
	for _, r := range Requirements(opts) {
		byName[r.Name] = r
	}
	want := map[string]Status{"git": Blocking, "Node.js 18+ and npm": Optional, "Engram": WillInstall,
		"CodeGraph": Optional, "OpenCode": Blocking, "Claude Code": Optional}
	for name, status := range want {
		if r, ok := byName[name]; !ok || r.Status != status {
			t.Errorf("%s: got %+v, want status %d", name, r, status)
		}
	}
	if len(Blockers(Requirements(opts))) != 2 {
		t.Error("git and OpenCode should block")
	}
	opts.Components = Components{}
	opts.Targets = []string{"claude"}
	for _, r := range Requirements(opts) {
		if r.Name == "Node.js 18+ and npm" || r.Name == "Engram" || r.Name == "CodeGraph" {
			t.Errorf("%s listed for a selection that does not need it", r.Name)
		}
	}

	// Install refuses to run with a blocker, but a dry run only warns.
	saved := checkRequirements
	checkRequirements = Requirements
	defer func() { checkRequirements = saved }()
	home := t.TempDir()
	o := testOptions(home)
	o.Targets = []string{"opencode"}
	if _, err := Install(o, testLogger{t}); err == nil || !strings.Contains(err.Error(), "OpenCode") {
		t.Errorf("install without OpenCode: %v", err)
	}
}

// Uninstall removes MCP servers from the Claude dir recorded at install, even
// when it is not passed again.
func TestUninstallClaudeMCPFromRecordedDir(t *testing.T) {
	home := t.TempDir()
	opts := testOptions(home)
	opts.Targets = []string{"claude"}
	if _, err := Install(opts, testLogger{t}); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(Options{Home: home}, testLogger{t}); err != nil {
		t.Fatal(err)
	}
	config, _ := readJSON(filepath.Join(opts.ClaudeDir, ".claude.json"))
	if servers, _ := config["mcpServers"].(map[string]any); len(servers) > 0 {
		t.Errorf("MCP servers left behind: %v", servers)
	}
}

func stubOpenCode(t *testing.T, run func(args ...string) (stdout, stderr string, err error)) {
	t.Helper()
	saved := runOpenCode
	t.Cleanup(func() { runOpenCode = saved })
	runOpenCode = func(_ context.Context, args ...string) ([]byte, []byte, error) {
		out, errOut, err := run(args...)
		return []byte(out), []byte(errOut), err
	}
}

func TestOpenCodeModelsFallsBackWithoutStandalone(t *testing.T) {
	var calls []string
	stubOpenCode(t, func(args ...string) (string, string, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(args) == 2 {
			return "", "error: unknown flag --standalone\n", errors.New("exit status 1")
		}
		// Windows line endings and colour codes.
		return "\x1b[1mopenai/gpt-5.5\x1b[0m\r\nopencode/big-pickle\r\n\r\n", "", nil
	})
	models, err := openCode{}.availableModels()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(models, ",") != "openai/gpt-5.5,opencode/big-pickle" {
		t.Errorf("models: %q", models)
	}
	if strings.Join(calls, " | ") != "models --standalone | models" {
		t.Errorf("calls: %v", calls)
	}
}

func TestOpenCodeModelsReason(t *testing.T) {
	stubOpenCode(t, func(args ...string) (string, string, error) {
		if len(args) == 2 {
			return "", "some log\nError: boom\n", errors.New("exit status 1")
		}
		return "", "", nil // succeeds but lists nothing
	})
	_, err := openCode{}.availableModels()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"opencode models --standalone: exit status 1: Error: boom", "opencode models: no models listed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("reason %q missing %q", err, want)
		}
	}
}

func TestSubagentMonitor(t *testing.T) {
	home := t.TempDir()
	opts := testOptions(home)
	opts.Targets = []string{"opencode"}
	tui := filepath.Join(opts.ConfigDir, "tui.json")
	if err := os.MkdirAll(opts.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	original := `{"theme": "tokyonight", "plugin": ["my-plugin", "opencode-subagent-statusline"]}`
	if err := os.WriteFile(tui, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	// Selected: left as Engram installs it.
	opts.Components.SubagentMonitor = true
	if _, err := Install(opts, testLogger{t}); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(tui); string(data) != original {
		t.Errorf("tui.json changed with the monitor selected: %s", data)
	}

	// Off (the default): removed, the rest kept, the original backed up.
	opts.Components.SubagentMonitor = false
	res, err := Install(opts, testLogger{t})
	if err != nil {
		t.Fatal(err)
	}
	config, _ := readJSON(tui)
	if plugins, _ := config["plugin"].([]any); len(plugins) != 1 || plugins[0] != "my-plugin" || config["theme"] != "tokyonight" {
		t.Errorf("tui.json after removal: %v", config)
	}
	backup := filepath.Join(res.BackupDir, filepath.FromSlash(strings.TrimPrefix(strings.Replace(slash(tui), ":", "", 1), "/")))
	if data, err := os.ReadFile(backup); err != nil || string(data) != original {
		t.Errorf("backup %s: %q, %v", backup, data, err)
	}

	// Only entry: the plugin list goes away.
	if err := os.WriteFile(tui, []byte(`{"plugin": ["opencode-subagent-statusline"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(opts, testLogger{t}); err != nil {
		t.Fatal(err)
	}
	if config, _ := readJSON(tui); config["plugin"] != nil {
		t.Errorf("empty plugin list left: %v", config)
	}
}

func TestSubagentMonitorV2(t *testing.T) {
	opts := testOptions(t.TempDir())
	opts.Targets = []string{"opencode"}
	cli := filepath.Join(opts.ConfigDir, "cli.json")
	if err := os.MkdirAll(opts.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	original := `{"$schema": "https://opencode.ai/v2/cli.json", "plugins": ["-opencode.notifications", "opencode-subagent-statusline@latest", {"package": "opencode-subagent-statusline"}]}`
	if err := os.WriteFile(cli, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(opts, testLogger{t}); err != nil {
		t.Fatal(err)
	}
	config, _ := readJSON(cli)
	plugins, ok := config["plugins"].([]any)
	if !ok || len(plugins) != 1 || plugins[0] != "-opencode.notifications" || config["$schema"] == nil {
		t.Errorf("cli.json after removal: %v", config)
	}

	// Only monitor entries: an empty list stays, the way OpenCode writes it.
	if err := os.WriteFile(cli, []byte(`{"plugins": ["opencode-subagent-statusline"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(opts, testLogger{t}); err != nil {
		t.Fatal(err)
	}
	if config, _ := readJSON(cli); config["plugins"] == nil || len(config["plugins"].([]any)) != 0 {
		t.Errorf("cli.json plugins: %v", config["plugins"])
	}
}
