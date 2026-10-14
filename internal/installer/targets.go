package installer

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/iesedobleac/agent-ecosystem/assets"
	"github.com/iesedobleac/agent-ecosystem/internal/render"
)

// ── OpenCode ─────────────────────────────────────────────────────────────────

func (r *run) installOpenCode(src *sources, st *TargetState) error {
	dir := r.opencodeConfigDir()
	values := r.values("opencode")
	r.runEngramSetup("opencode")

	files := map[string]string{}
	for _, a := range src.agents {
		text, err := render.OpenCodeAgent(a, values, r.opts.Models["opencode"])
		if err != nil {
			return err
		}
		files["agents/"+a.Name+".md"] = text
	}
	skills, err := r.skillFiles("opencode", values)
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

	rules, err := r.rules("opencode", values)
	if err != nil {
		return err
	}
	if err := r.upsertBlock(filepath.Join(dir, "AGENTS.md"), rules); err != nil {
		return err
	}
	r.log.OK("OpenCode: rules in " + filepath.Join(dir, "AGENTS.md"))

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

func (r *run) uninstallOpenCode(st *TargetState) error {
	dir := st.Dir
	if dir == "" {
		dir = r.opencodeConfigDir()
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

// ── Claude Code ──────────────────────────────────────────────────────────────

func (r *run) claude(args ...string) error {
	cmd := exec.Command("claude", args...)
	cmd.Env = append(os.Environ(), "CLAUDE_CONFIG_DIR="+r.claudeDir())
	return cmd.Run()
}

func (r *run) claudeMCPExists(name string) bool {
	if r.bins["claude"] != "" {
		return r.claude("mcp", "get", name) == nil
	}
	config, err := readJSON(r.claudeJSON())
	if err != nil {
		return false
	}
	servers, _ := config["mcpServers"].(map[string]any)
	_, ok := servers[name]
	return ok
}

func (r *run) claudeMCPAdd(name string, entry map[string]any) error {
	if r.opts.DryRun {
		r.log.Info("[dry-run] Claude Code MCP " + name)
		return nil
	}
	if r.bins["claude"] != "" {
		data, _ := json.Marshal(entry)
		if r.claude("mcp", "add-json", "--scope", "user", name, string(data)) == nil {
			return nil
		}
		r.log.Warn("claude mcp add-json " + name + " failed; writing " + r.claudeJSON() + " instead")
	}
	file := r.claudeJSON()
	config, err := readJSON(file)
	if err != nil {
		return err
	}
	object(config, "mcpServers")[name] = entry
	r.backupFile(file)
	return r.writeJSON(file, config)
}

func (r *run) claudeMCPRemove(name string) error {
	if r.opts.DryRun {
		r.log.Info("[dry-run] remove Claude Code MCP " + name)
		return nil
	}
	if r.bins["claude"] != "" && r.claude("mcp", "remove", "--scope", "user", name) == nil {
		return nil
	}
	file := r.claudeJSON()
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

func (r *run) installClaude(src *sources, st *TargetState) error {
	dir := r.claudeDir()
	values := r.values("claude")
	if r.bins["claude"] != "" {
		r.runEngramSetup("claude-code", "CLAUDE_CONFIG_DIR="+dir)
	}
	_, serverNames, err := r.servers(values)
	if err != nil {
		return err
	}
	files := map[string]string{}
	for _, a := range src.agents {
		text, err := render.ClaudeAgent(a, src.agents, values, r.opts.Models["claude"], serverNames)
		if err != nil {
			return err
		}
		files["agents/"+a.Name+".md"] = text
	}
	skills, err := r.skillFiles("claude", values)
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

	rules, err := r.rules("claude", values)
	if err != nil {
		return err
	}
	if err := r.upsertBlock(filepath.Join(dir, "CLAUDE.md"), rules); err != nil {
		return err
	}
	r.log.OK("Claude Code: rules in " + filepath.Join(dir, "CLAUDE.md"))

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

	wanted, _, err := r.servers(values)
	if err != nil {
		return err
	}
	var added []string
	for _, name := range sortedKeys(wanted) {
		if !contains(st.MCP, name) && r.claudeMCPExists(name) {
			r.log.Info("Claude Code: MCP " + name + " already configured; left as is")
			continue
		}
		if err := r.claudeMCPAdd(name, wanted[name].ClaudeEntry()); err != nil {
			return err
		}
		added = append(added, name)
	}
	var owned []string
	for _, name := range st.MCP {
		if _, ok := wanted[name]; ok {
			owned = append(owned, name)
		} else if err := r.claudeMCPRemove(name); err != nil {
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

func (r *run) uninstallClaude(st *TargetState) error {
	dir := st.Dir
	if dir == "" {
		dir = r.claudeDir()
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
		if err := r.claudeMCPRemove(name); err != nil {
			return err
		}
	}
	r.log.OK("Claude Code: removed from " + dir)
	return nil
}

// ── Shell profile ────────────────────────────────────────────────────────────

func (r *run) profileFile() string {
	switch filepath.Base(os.Getenv("SHELL")) {
	case "zsh":
		return filepath.Join(r.home, ".zshrc")
	case "bash":
		if runtime.GOOS == "darwin" {
			return filepath.Join(r.home, ".bash_profile")
		}
		return filepath.Join(r.home, ".bashrc")
	}
	return filepath.Join(r.home, ".profile")
}

func (r *run) installProfile() (string, error) {
	if r.opts.NoShellProfile || !r.opts.Components.Engram {
		return "", nil
	}
	if runtime.GOOS == "windows" {
		if !r.opts.DryRun {
			_ = exec.Command("setx", "ENGRAM_NUDGE_COOLDOWN_SECS", "86400").Run()
		}
		r.log.OK("ENGRAM_NUDGE_COOLDOWN_SECS=86400 set for the user (setx)")
		return "setx", nil
	}
	file := r.profileFile()
	current, _ := os.ReadFile(file)
	if !strings.Contains(string(current), profileMark) {
		text := strings.TrimRight(string(current), "\n") + "\n\n" + profileMark +
			": at most one Engram memory reminder per session per day\n" + profileLine + "\n"
		if err := r.write(file, text); err != nil {
			return "", err
		}
	}
	r.log.OK("ENGRAM_NUDGE_COOLDOWN_SECS=86400 in " + file)
	return file, nil
}

var profileRe = regexp.MustCompile(`\n*` + regexp.QuoteMeta(profileMark) + `.*\n` + regexp.QuoteMeta(profileLine) + `\n?`)

func (r *run) removeProfile(file string) error {
	if file == "" || file == "setx" {
		return nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	return r.write(file, profileRe.ReplaceAllString(string(data), "\n"))
}

// pruneInstalled removes directories left empty after files were removed.
func (r *run) pruneInstalled(dir string, subdirs ...string) {
	if r.opts.DryRun {
		return
	}
	for _, sub := range subdirs {
		entries, _ := os.ReadDir(filepath.Join(dir, sub))
		for _, e := range entries {
			if e.IsDir() {
				pruneEmpty(filepath.Join(dir, sub, e.Name()))
			}
		}
	}
}

func listOr(items []string, empty string) string {
	if len(items) == 0 {
		return empty
	}
	return strings.Join(items, ", ")
}
