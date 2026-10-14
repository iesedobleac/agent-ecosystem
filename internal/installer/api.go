package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetectTargets returns the assistants found on this machine (OpenCode if none).
func DetectTargets(opts Options) []string {
	r, err := newRun(opts, nopLogger{})
	if err != nil {
		return []string{"opencode"}
	}
	var found []string
	if _, err := os.Stat(r.opencodeConfigDir()); which("opencode") != "" || err == nil {
		found = append(found, "opencode")
	}
	if _, err := os.Stat(r.claudeDir()); which("claude") != "" || err == nil {
		found = append(found, "claude")
	}
	if len(found) == 0 {
		return []string{"opencode"}
	}
	return found
}

// Install installs or updates the selection for each target.
func Install(opts Options, log Logger) (*Result, error) {
	if len(opts.Targets) == 0 {
		opts.Targets = DetectTargets(opts)
	}
	for _, t := range opts.Targets {
		if !contains(AllTargets, t) {
			return nil, fmt.Errorf("unknown target %q (supported: %s)", t, strings.Join(AllTargets, ", "))
		}
	}
	if blockers := Blockers(checkRequirements(opts)); len(blockers) > 0 {
		var names []string
		for _, b := range blockers {
			names = append(names, b.Name+" ("+b.Hint+")")
		}
		msg := "missing required tools: " + strings.Join(names, "; ")
		if !opts.DryRun {
			return nil, errors.New(msg)
		}
		log.Warn(msg)
	}
	r, err := newRun(opts, log)
	if err != nil {
		return nil, err
	}
	if err := r.ensureDependencies(opts.Targets); err != nil {
		return nil, err
	}
	src, err := r.loadSources()
	if err != nil {
		return nil, err
	}
	if !r.opts.DryRun {
		deliverables := expandHome(r.opts.DeliverablesDir, r.home)
		if err := os.MkdirAll(deliverables, 0o755); err != nil {
			return nil, err
		}
	}
	state := r.readState()
	for _, t := range opts.Targets {
		st := state.Targets[t]
		if st == nil {
			st = &TargetState{}
		}
		switch t {
		case "opencode":
			err = r.installOpenCode(src, st)
		case "claude":
			err = r.installClaude(src, st)
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", TargetLabels[t], err)
		}
		st.Version, st.Installed = Version, r.stamp
		state.Targets[t] = st
	}
	profile, err := r.installProfile()
	if err != nil {
		return nil, err
	}
	if profile != "" {
		state.Profile = profile
	}
	selection := r.opts
	state.Selection = &selection
	if err := r.writeState(state); err != nil {
		return nil, err
	}

	res := &Result{Targets: opts.Targets}
	if contains(opts.Targets, "opencode") {
		res.NextSteps = append(res.NextSteps, "OpenCode: restart it (and `opencode service restart` from a new terminal), connect your provider with /connect, and pick sdd-orchestrator or android-orchestrator.")
	}
	if contains(opts.Targets, "claude") {
		res.NextSteps = append(res.NextSteps, "Claude Code: start an orchestrator with `claude --agent sdd-orchestrator` or `claude --agent android-orchestrator`.")
	}
	if r.opts.Components.CodeGraph {
		res.NextSteps = append(res.NextSteps, "Index each repository once: codegraph init (in the repository root).")
	}
	if _, err := os.Stat(r.backup); err == nil {
		res.BackupDir = r.backup
	}
	return res, nil
}

// Uninstall removes what previous installs added, for opts.Targets or every installed target.
func Uninstall(opts Options, log Logger) error {
	r, err := newRun(opts, log)
	if err != nil {
		return err
	}
	for _, name := range []string{"claude"} {
		r.bins[name] = which(name)
	}
	state := r.readState()
	targets := opts.Targets
	if len(targets) == 0 {
		targets = sortedKeys(state.Targets)
	}
	if len(targets) == 0 {
		return fmt.Errorf("nothing to uninstall")
	}
	for _, t := range targets {
		st := state.Targets[t]
		if st == nil {
			log.Warn(TargetLabels[t] + " is not installed")
			continue
		}
		switch t {
		case "opencode":
			err = r.uninstallOpenCode(st)
		case "claude":
			err = r.uninstallClaude(st)
		}
		if err != nil {
			return err
		}
		delete(state.Targets, t)
	}
	if len(state.Targets) == 0 {
		if err := r.removeProfile(state.Profile); err != nil {
			return err
		}
		state.Profile, state.Selection = "", nil
	}
	if err := r.writeState(state); err != nil {
		return err
	}
	if _, err := os.Stat(r.backup); err == nil {
		log.OK("Backups: " + r.backup)
	}
	log.Info("Engram, CodeGraph and the assistants themselves were not removed.")
	return nil
}

// Doctor runs read-only checks. It returns false when a check fails.
func Doctor(opts Options, log Logger) bool {
	r, err := newRun(opts, log)
	if err != nil {
		log.Warn(err.Error())
		return false
	}
	ok := true
	data, err := os.ReadFile(r.stateFile())
	var state State
	switch {
	case os.IsNotExist(err):
		log.Warn("No install recorded in " + r.stateFile())
	case err != nil:
		log.Warn("Cannot read " + r.stateFile() + ": " + err.Error())
		ok = false
	case json.Unmarshal(data, &state) != nil:
		log.Warn(r.stateFile() + " is not valid JSON; restore a backup or repair it")
		ok = false
	default:
		log.OK("State: " + r.stateFile())
	}
	targets := sortedKeys(state.Targets)
	comps := DefaultComponents()
	if state.Selection != nil {
		comps = state.Selection.Components
	}
	for _, req := range Requirements(Options{Targets: targets, Components: comps}) {
		switch req.Status {
		case Present:
			log.OK(req.Name + " " + req.Detail)
		case WillInstall:
			log.Warn(req.Name + " missing: `agent-ecosystem sync` installs it " + req.Detail)
		default:
			log.Warn(req.Name + " missing: " + req.Hint)
			ok = ok && req.Status != Blocking
		}
	}
	for _, t := range targets {
		st := state.Targets[t]
		missing, changed := 0, 0
		for rel, hash := range st.Files {
			data, err := os.ReadFile(filepath.Join(st.Dir, filepath.FromSlash(rel)))
			if err != nil {
				missing++
			} else if sum(string(data)) != hash {
				changed++
			}
		}
		msg := fmt.Sprintf("%s: %d files in %s (version %s)", TargetLabels[t], len(st.Files), st.Dir, st.Version)
		switch {
		case missing > 0:
			log.Warn(fmt.Sprintf("%s, %d missing; run `agent-ecosystem sync`", msg, missing))
			ok = false
		case changed > 0:
			log.Warn(fmt.Sprintf("%s, %d edited by hand (sync backs them up before replacing)", msg, changed))
		default:
			log.OK(msg)
		}
	}
	return ok
}

type nopLogger struct{}

func (nopLogger) Info(string) {}
func (nopLogger) OK(string)   {}
func (nopLogger) Warn(string) {}
