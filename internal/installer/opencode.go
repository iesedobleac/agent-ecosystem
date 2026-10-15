package installer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/iesedobleac/iesedoble-ai/assets"
	"github.com/iesedobleac/iesedoble-ai/internal/render"
)

type openCode struct{}

func (openCode) id() string     { return "opencode" }
func (openCode) label() string  { return "OpenCode" }
func (openCode) binary() string { return "opencode" }

// flagPrefix is empty: OpenCode was the first target and keeps --planner.
func (openCode) flagPrefix() string { return "" }
func (openCode) steps() int         { return 4 }

func (openCode) defaultModels() render.Models {
	return render.Models{"planner": "github-copilot/claude-opus-5.5", "analyst": "github-copilot/claude-sonnet-5.5",
		"executor": "github-copilot/claude-sonnet-5.5", "scout": "github-copilot/claude-haiku-5.5",
		"reviewer": "github-copilot/gpt-5.6-terra"}
}

// defaultEfforts is nil: OpenCode sets reasoning per model variant, whose
// names depend on each model; a variant can go in the model ID (model#high).
func (openCode) defaultEfforts() render.Efforts { return nil }

func (openCode) nextStep() string {
	return "OpenCode: restart it (and `opencode service restart` from a new terminal), connect your provider with /connect, and pick sdd-orchestrator or android-orchestrator."
}

// path asks OpenCode for one of its directories ("config", "data", "tmp").
func (openCode) path(name string) string {
	if which("opencode") == "" {
		return ""
	}
	out, err := command("opencode", "debug", "paths", name)
	if err != nil || out == "" || strings.Contains(out, "\n") {
		return ""
	}
	return out
}

func (o openCode) configDir(r *run) string {
	if r.opts.ConfigDir != "" {
		return filepath.Clean(expandHome(r.opts.ConfigDir, r.home))
	}
	if p := o.path("config"); p != "" {
		return p
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(r.home, ".config")
	}
	return filepath.Join(base, "opencode")
}

func (o openCode) detected(r *run) bool {
	_, err := os.Stat(o.configDir(r))
	return which("opencode") != "" || err == nil
}

func (o openCode) dirs(r *run) targetDirs {
	data := o.path("data")
	if data == "" {
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(r.home, ".local/share")
		}
		data = filepath.Join(base, "opencode")
	}
	return targetDirs{tmp: tempDir(o.path("tmp"), "opencode"), data: data, skills: filepath.Join(o.configDir(r), "skills")}
}

func (openCode) requirements(opts Options) []Requirement {
	r := Requirement{Name: "OpenCode", Purpose: "selected assistant"}
	if v, ok := version("opencode", "--version"); ok {
		r.Detail = v
	} else {
		r.Status, r.Hint = Blocking, "install it from https://opencode.ai/docs and run again"
	}
	reqs := []Requirement{r}
	if runtime.GOOS == "windows" && !opts.NoGitBash {
		gb := Requirement{Name: "Git Bash", Purpose: "the agents' shell commands need a POSIX shell"}
		if p := gitBash(); p != "" {
			gb.Detail = p
		} else {
			gb.Status, gb.Hint = Optional, "winget install Git.Git"
		}
		reqs = append(reqs, gb)
	}
	return reqs
}

// opencodeModelsArgs are tried in order: --standalone lists from a private
// server, since the background service can be stopped and then lists nothing;
// versions without that flag list with the plain command.
var opencodeModelsArgs = [][]string{{"models", "--standalone"}, {"models"}}

// runOpenCode runs opencode with args; replaced in tests.
var runOpenCode = func(ctx context.Context, args ...string) (stdout, stderr []byte, err error) {
	cmd := exec.CommandContext(ctx, "opencode", args...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	// On Windows opencode is often a .cmd shim whose child process keeps the
	// pipes open after a timeout kill; stop waiting for them.
	cmd.WaitDelay = 2 * time.Second
	err = cmd.Run()
	return out.Bytes(), errOut.Bytes(), err
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func (openCode) availableModels() ([]string, error) {
	var reasons []string
	for _, args := range opencodeModelsArgs {
		models, err := listOpenCodeModels(args...)
		if err == nil {
			return models, nil
		}
		reasons = append(reasons, "opencode "+strings.Join(args, " ")+": "+err.Error())
	}
	return nil, errors.New(strings.Join(reasons, "; "))
}

func listOpenCodeModels(args ...string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, stderr, err := runOpenCode(ctx, args...)
	switch {
	case ctx.Err() == context.DeadlineExceeded:
		return nil, errors.New("timed out after 60s")
	case err != nil:
		if msg := strings.TrimSpace(ansiRe.ReplaceAllString(lastLine(string(stderr)), "")); msg != "" {
			return nil, fmt.Errorf("%v: %s", err, msg)
		}
		return nil, err
	}
	var models []string
	for _, line := range strings.Split(ansiRe.ReplaceAllString(string(out), ""), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "/") && !strings.ContainsAny(line, " \t") {
			models = append(models, line)
		}
	}
	if len(models) == 0 {
		return nil, errors.New("no models listed (is a provider connected?)")
	}
	return models, nil
}

func (openCode) knownModel(model string, available []string) bool {
	return contains(available, model)
}

func (o openCode) install(r *run, src *sources, st *TargetState) error {
	dir := o.configDir(r)
	values := r.values(o)
	r.step("OpenCode: memory")
	r.runEngramSetup("opencode")
	if err := o.syncSubagentMonitor(r); err != nil {
		return err
	}
	r.step("OpenCode: agents, skills and plugin")

	files := map[string]string{}
	for _, a := range src.agents {
		text, err := render.OpenCodeAgent(a, values, r.opts.Models[o.id()])
		if err != nil {
			return err
		}
		files["agents/"+a.Name+".md"] = text
	}
	skills, err := r.skillFiles(o.id(), values)
	if err != nil {
		return err
	}
	for k, v := range skills {
		files[k] = v
	}
	if r.opts.Components.Plugin {
		err := fs.WalkDir(assets.FS, "plugins/opencode", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, err := fs.ReadFile(assets.FS, p)
			if err != nil {
				return err
			}
			files["plugins/"+strings.TrimPrefix(p, "plugins/opencode/")] = string(data)
			return nil
		})
		if err != nil {
			return err
		}
	}
	if st.Files, err = r.syncFiles(dir, files, st.Files); err != nil {
		return err
	}
	r.pruneInstalled(dir, "agents", "skills", "plugins")
	r.log.OK(fmt.Sprintf("OpenCode: %d files in %s", len(st.Files), dir))

	r.step("OpenCode: rules")
	rules, err := r.rules(o.id(), values)
	if err != nil {
		return err
	}
	if err := r.upsertBlock(filepath.Join(dir, "AGENTS.md"), rules); err != nil {
		return err
	}
	r.log.OK("OpenCode: rules in " + filepath.Join(dir, "AGENTS.md"))

	r.step("OpenCode: config and MCP servers")
	file := filepath.Join(dir, "opencode.json")
	if _, err := os.Stat(filepath.Join(dir, "opencode.jsonc")); err == nil {
		file = filepath.Join(dir, "opencode.jsonc")
	}
	raw, _ := os.ReadFile(file)
	config, err := readJSON(file)
	if err != nil {
		return err
	}
	if _, ok := config["$schema"]; !ok {
		config["$schema"] = "https://opencode.ai/config.json"
	}
	mcp := object(config, "mcp")
	// Keep the shape the file already uses: legacy servers under "mcp", V2 under "mcp.servers".
	legacy := false
	for k := range mcp {
		if k != "servers" && k != "timeout" {
			legacy = true
		}
	}
	servers := mcp
	if !legacy {
		servers = object(mcp, "servers")
	}
	wanted, _, err := r.servers(values)
	if err != nil {
		return err
	}
	var added []string
	for _, name := range sortedKeys(wanted) {
		if _, exists := servers[name]; exists && !contains(st.MCP, name) {
			r.log.Info("OpenCode: MCP " + name + " already configured; left as is")
			continue
		}
		servers[name] = wanted[name].OpenCodeEntry(legacy)
		added = append(added, name)
	}
	var owned []string
	for _, name := range st.MCP {
		if _, ok := wanted[name]; ok {
			owned = append(owned, name)
		} else {
			delete(servers, name)
		}
	}
	for _, name := range added {
		if !contains(owned, name) {
			owned = append(owned, name)
		}
	}
	st.MCP = owned
	if runtime.GOOS == "windows" && !r.opts.NoGitBash {
		if _, ok := config["shell"]; !ok {
			if bash := gitBash(); bash != "" {
				config["shell"] = bash
				r.log.Info("OpenCode: shell set to Git Bash (" + bash + ")")
			}
			if _, ok := config["shell"]; !ok {
				r.log.Warn("Git Bash not found: agents expect a POSIX shell (tail, pipes); install Git for Windows")
			}
		}
	}
	if hasComments(string(raw)) {
		r.log.Warn(filepath.Base(file) + " is rewritten as plain JSON; comments are not kept (backup in " + r.backup + ")")
	}
	r.backupFile(file)
	if err := r.writeJSON(file, config); err != nil {
		return err
	}
	st.Dir = dir
	r.log.OK("OpenCode: MCP servers " + listOr(added, "unchanged"))
	return nil
}

// subagentMonitor is the OpenCode TUI plugin `engram setup opencode` adds.
const subagentMonitor = "opencode-subagent-statusline"

// isSubagentMonitor matches a plugin entry: the package name, optionally with a
// version ("…@latest"), or V2's {"package": …} form.
func isSubagentMonitor(entry any) bool {
	if obj, ok := entry.(map[string]any); ok {
		entry = obj["package"]
	}
	name, _ := entry.(string)
	return name == subagentMonitor || strings.HasPrefix(name, subagentMonitor+"@")
}

// syncSubagentMonitor removes the subagent monitor unless it is selected.
// `engram setup` adds it to the V1 tui.json on every run, so this runs after it;
// OpenCode V2 migrates tui.json into cli.json and loads plugins from there.
func (o openCode) syncSubagentMonitor(r *run) error {
	if r.opts.Components.SubagentMonitor {
		return nil
	}
	dir := o.configDir(r)
	for _, f := range []struct {
		names     []string
		key       string
		dropEmpty bool
	}{
		{[]string{"tui.jsonc", "tui.json"}, "plugin", true},   // V1
		{[]string{"cli.jsonc", "cli.json"}, "plugins", false}, // V2
	} {
		for _, name := range f.names {
			file := filepath.Join(dir, name)
			if _, err := os.Stat(file); err != nil {
				continue
			}
			if err := r.removePluginEntry(file, f.key, f.dropEmpty); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// removePluginEntry drops the subagent monitor from the key list in file.
func (r *run) removePluginEntry(file, key string, dropEmpty bool) error {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	config, err := readJSON(file)
	if err != nil {
		return err
	}
	plugins, _ := config[key].([]any)
	kept := []any{}
	for _, p := range plugins {
		if !isSubagentMonitor(p) {
			kept = append(kept, p)
		}
	}
	if len(kept) == len(plugins) {
		return nil
	}
	if len(kept) == 0 && dropEmpty {
		delete(config, key)
	} else {
		config[key] = kept
	}
	if hasComments(string(raw)) {
		r.log.Warn(filepath.Base(file) + " is rewritten as plain JSON; comments are not kept (backup in " + r.backup + ")")
	}
	r.backupFile(file)
	if err := r.writeJSON(file, config); err != nil {
		return err
	}
	r.log.OK("OpenCode: subagent monitor removed from " + file)
	return nil
}

func (o openCode) uninstall(r *run, st *TargetState) error {
	dir := st.Dir
	if dir == "" {
		dir = o.configDir(r)
	}
	for rel := range st.Files {
		r.remove(filepath.Join(dir, filepath.FromSlash(rel)))
	}
	if !r.opts.DryRun {
		for _, sub := range []string{"agents", "skills", "plugins"} {
			pruneEmpty(filepath.Join(dir, sub))
		}
	}
	if err := r.removeBlock(filepath.Join(dir, "AGENTS.md")); err != nil {
		return err
	}
	for _, name := range []string{"opencode.jsonc", "opencode.json"} {
		file := filepath.Join(dir, name)
		if _, err := os.Stat(file); err != nil {
			continue
		}
		config, err := readJSON(file)
		if err != nil {
			return err
		}
		if mcp, ok := config["mcp"].(map[string]any); ok {
			for _, s := range st.MCP {
				delete(mcp, s)
				if servers, ok := mcp["servers"].(map[string]any); ok {
					delete(servers, s)
				}
			}
		}
		r.backupFile(file)
		if err := r.writeJSON(file, config); err != nil {
			return err
		}
		break
	}
	r.log.OK("OpenCode: removed from " + dir)
	return nil
}
