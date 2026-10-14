package installer

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ClaudeModels are the values a Claude Code subagent's model field takes besides
// a full model ID.
var ClaudeModels = []string{"opus", "sonnet", "haiku", "fable", "inherit"}

// AvailableModels lists the models a target can use. ok is false when they
// cannot be listed: OpenCode is missing, fails or has no connected provider.
func AvailableModels(target string) (models []string, ok bool) {
	switch target {
	case "claude":
		return ClaudeModels, true
	case "opencode":
		if which("opencode") == "" {
			return nil, false
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		// --standalone: the background service can be stopped and then lists nothing.
		out, err := exec.CommandContext(ctx, "opencode", "models", "--standalone").Output()
		if err != nil {
			return nil, false
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "/") && !strings.ContainsAny(line, " \t") {
				models = append(models, line)
			}
		}
		return models, len(models) > 0
	}
	return nil, false
}

// listModels is replaced in tests, which run without the assistants installed.
var listModels = AvailableModels

// KnownModel reports whether model is among the available ones. Claude Code
// also takes full model IDs.
func KnownModel(target, model string, available []string) bool {
	if target == "claude" && strings.HasPrefix(model, "claude-") {
		return true
	}
	return contains(available, model)
}

// checkModels warns about models the assistant does not list; the install goes on.
func (r *run) checkModels() {
	for _, t := range r.opts.Targets {
		available, ok := listModels(t)
		if !ok {
			r.log.Warn(TargetLabels[t] + ": could not list the available models (is a provider connected?); the chosen ones are not checked")
			continue
		}
		for _, p := range []string{"planner", "executor", "reviewer"} {
			if m := r.opts.Models[t][p]; !KnownModel(t, m, available) {
				r.log.Warn(fmt.Sprintf("%s: the %s model %q is not among the available models", TargetLabels[t], p, m))
			}
		}
	}
}
