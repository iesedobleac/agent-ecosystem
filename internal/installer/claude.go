package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/iesedobleac/agent-ecosystem/internal/render"
)

type claudeCode struct{}

func (claudeCode) id() string         { return "claude" }
func (claudeCode) label() string      { return "Claude Code" }
func (claudeCode) binary() string     { return "claude" }
func (claudeCode) flagPrefix() string { return "claude-" }
func (claudeCode) steps() int         { return 5 }

func (claudeCode) defaultModels() render.Models {
	return render.Models{"planner": "opus", "executor": "haiku", "reviewer": "sonnet"}
}

func (claudeCode) nextStep() string {
	return "Claude Code: start an orchestrator with `claude --agent sdd-orchestrator` or `claude --agent android-orchestrator`."
}

// ClaudeModels are the values a Claude Code subagent's model field takes besides
// a full model ID.
var ClaudeModels = []string{"opus", "sonnet", "haiku", "fable", "inherit"}

// Session-wide deny rules: Claude Code has no per-agent shell permissions.
var claudeDeny = []string{"Bash(git push)", "Bash(git push *)", "Bash(git reset --hard)", "Bash(git reset --hard *)",
	"Bash(git clean *)", "Bash(git rebase *)"}

func (claudeCode) dir(r *run) string {
	if r.opts.ClaudeDir != "" {
		return filepath.Clean(expandHome(r.opts.ClaudeDir, r.home))
	}
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return d
	}
	return filepath.Join(r.home, ".claude")
}

// configJSON is the .claude.json of a config dir: with a custom one
// (CLAUDE_CONFIG_DIR), Claude Code keeps it inside that dir.
func (claudeCode) configJSON(r *run, dir string) string {
	if dir != filepath.Join(r.home, ".claude") {
		return filepath.Join(dir, ".claude.json")
	}
	return filepath.Join(r.home, ".claude.json")
}

func (c claudeCode) detected(r *run) bool {
	_, err := os.Stat(c.dir(r))
	return which("claude") != "" || err == nil
}

func (c claudeCode) dirs(r *run) targetDirs {
	dir := c.dir(r)
	return targetDirs{tmp: tempDir("", "agent-ecosystem"), data: dir, skills: filepath.Join(dir, "skills")}
}

func (claudeCode) requirements(Options) []Requirement {
	r := Requirement{Name: "Claude Code", Purpose: "selected assistant"}
	if v, ok := version("claude", "--version"); ok {
		r.Detail = v
	} else {
		r.Status, r.Hint = Optional, "its configuration is prepared anyway; install it from https://docs.claude.com/en/docs/claude-code/setup"
	}
	return []Requirement{r}
}

// availableModels has no command to call: Claude Code takes a fixed set of aliases.
func (claudeCode) availableModels() ([]string, error) { return ClaudeModels, nil }

// knownModel also accepts full model IDs.
func (claudeCode) knownModel(model string, available []string) bool {
	return strings.HasPrefix(model, "claude-") || contains(available, model)
}

// cli runs the claude command against the config dir dir.
func (claudeCode) cli(dir string, args ...string) error {
	cmd := exec.Command("claude", args...)
	cmd.Env = append(os.Environ(), "CLAUDE_CONFIG_DIR="+dir)
	return cmd.Run()
}

func (c claudeCode) mcpExists(r *run, dir, name string) bool {
	if r.bins["claude"] != "" {
		return c.cli(dir, "mcp", "get", name) == nil
	}
	config, err := readJSON(c.configJSON(r, dir))
	if err != nil {
		return false
	}
	servers, _ := config["mcpServers"].(map[string]any)
	_, ok := servers[name]
	return ok
}

func (c claudeCode) mcpAdd(r *run, dir, name string, entry map[string]any) error {
	if r.opts.DryRun {
		r.log.Info("[dry-run] Claude Code MCP " + name)
		return nil
	}
	if r.bins["claude"] != "" {
		data, _ := json.Marshal(entry)
		if c.cli(dir, "mcp", "add-json", "--scope", "user", name, string(data)) == nil {
			return nil
		}
		r.log.Warn("claude mcp add-json " + name + " failed; writing " + c.configJSON(r, dir) + " instead")
	}
	file := c.configJSON(r, dir)
	config, err := readJSON(file)
	if err != nil {
		return err
	}
	object(config, "mcpServers")[name] = entry
	r.backupFile(file)
	return r.writeJSON(file, config)
}

func (c claudeCode) mcpRemove(r *run, dir, name string) error {
	if r.opts.DryRun {
		r.log.Info("[dry-run] remove Claude Code MCP " + name)
		return nil
	}
	if r.bins["claude"] != "" && c.cli(dir, "mcp", "remove", "--scope", "user", name) == nil {
		return nil
	}
	file := c.configJSON(r, dir)
	config, err := readJSON(file)
	if err != nil {
		return err
	}
	if servers, ok := config["mcpServers"].(map[string]any); ok {
		if _, ok := servers[name]; ok {
			delete(servers, name)
			r.backupFile(file)
			return r.writeJSON(file, config)
		}
	}
	return nil
}

func (c claudeCode) install(r *run, src *sources, st *TargetState) error {
	dir := c.dir(r)
	values := r.values(c)
	r.step("Claude Code: memory")
	if r.bins["claude"] != "" {
		r.runEngramSetup("claude-code", "CLAUDE_CONFIG_DIR="+dir)
	}
	r.step("Claude Code: agents and skills")
	_, serverNames, err := r.servers(values)
	if err != nil {
		return err
	}
	files := map[string]string{}
	for _, a := range src.agents {
		text, err := render.ClaudeAgent(a, src.agents, values, r.opts.Models[c.id()], serverNames)
		if err != nil {
			return err
		}
		files["agents/"+a.Name+".md"] = text
	}
	skills, err := r.skillFiles(c.id(), values)
	if err != nil {
		return err
	}
	for k, v := range skills {
		files[k] = v
	}
	if st.Files, err = r.syncFiles(dir, files, st.Files); err != nil {
		return err
	}
	r.pruneInstalled(dir, "agents", "skills")
	r.log.OK(fmt.Sprintf("Claude Code: %d files in %s", len(st.Files), dir))

	r.step("Claude Code: rules")
	rules, err := r.rules(c.id(), values)
	if err != nil {
		return err
	}
	if err := r.upsertBlock(filepath.Join(dir, "CLAUDE.md"), rules); err != nil {
		return err
	}
	r.log.OK("Claude Code: rules in " + filepath.Join(dir, "CLAUDE.md"))

	r.step("Claude Code: session deny rules")
	settingsFile := filepath.Join(dir, "settings.json")
	settings, err := readJSON(settingsFile)
	if err != nil {
		return err
	}
	perms := object(settings, "permissions")
	var deny []any
	if d, ok := perms["deny"].([]any); ok {
		deny = d
	}
	var addedDeny []string
	for _, rule := range claudeDeny {
		found := false
		for _, d := range deny {
			if d == rule {
				found = true
			}
		}
		if !found {
			deny = append(deny, rule)
			addedDeny = append(addedDeny, rule)
		}
	}
	if len(addedDeny) > 0 {
		perms["deny"] = deny
		r.backupFile(settingsFile)
		if err := r.writeJSON(settingsFile, settings); err != nil {
			return err
		}
	}
	for _, d := range addedDeny {
		if !contains(st.Deny, d) {
			st.Deny = append(st.Deny, d)
		}
	}
	r.log.OK("Claude Code: session deny rules " + listOr(addedDeny, "already present"))

	r.step("Claude Code: MCP servers")
	wanted, _, err := r.servers(values)
	if err != nil {
		return err
	}
	var added []string
	for _, name := range sortedKeys(wanted) {
		if c.mcpExists(r, dir, name) {
			if !contains(st.MCP, name) {
				r.log.Info("Claude Code: MCP " + name + " already configured; left as is")
				continue
			}
			// Ours from a previous run: `claude mcp add-json` does not overwrite,
			// so remove it and add the current entry.
			if err := c.mcpRemove(r, dir, name); err != nil {
				return err
			}
		}
		if err := c.mcpAdd(r, dir, name, wanted[name].ClaudeEntry()); err != nil {
			return err
		}
		added = append(added, name)
	}
	var owned []string
	for _, name := range st.MCP {
		if _, ok := wanted[name]; ok {
			owned = append(owned, name)
		} else if err := c.mcpRemove(r, dir, name); err != nil {
			return err
		}
	}
	for _, name := range added {
		if !contains(owned, name) {
			owned = append(owned, name)
		}
	}
	st.MCP = owned
	st.Dir = dir
	r.log.OK("Claude Code: MCP servers " + listOr(added, "unchanged"))
	return nil
}

func (c claudeCode) uninstall(r *run, st *TargetState) error {
	dir := st.Dir
	if dir == "" {
		dir = c.dir(r)
	}
	for rel := range st.Files {
		r.remove(filepath.Join(dir, filepath.FromSlash(rel)))
	}
	if !r.opts.DryRun {
		for _, sub := range []string{"agents", "skills"} {
			pruneEmpty(filepath.Join(dir, sub))
		}
	}
	if err := r.removeBlock(filepath.Join(dir, "CLAUDE.md")); err != nil {
		return err
	}
	settingsFile := filepath.Join(dir, "settings.json")
	if len(st.Deny) > 0 {
		if settings, err := readJSON(settingsFile); err == nil {
			if perms, ok := settings["permissions"].(map[string]any); ok {
				var kept []any
				deny, _ := perms["deny"].([]any)
				for _, d := range deny {
					if s, ok := d.(string); !ok || !contains(st.Deny, s) {
						kept = append(kept, d)
					}
				}
				perms["deny"] = kept
				r.backupFile(settingsFile)
				if err := r.writeJSON(settingsFile, settings); err != nil {
					return err
				}
			}
		}
	}
	for _, name := range st.MCP {
		if err := c.mcpRemove(r, dir, name); err != nil {
			return err
		}
	}
	r.log.OK("Claude Code: removed from " + dir)
	return nil
}
