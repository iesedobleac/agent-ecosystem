package installer

import "fmt"

// AvailableModels lists the models a target can use. ok is false when they
// cannot be listed: OpenCode is missing, fails or has no connected provider.
func AvailableModels(target string) (models []string, ok bool) {
	if t := targetByID(target); t != nil {
		return t.availableModels()
	}
	return nil, false
}

// listModels is replaced in tests, which run without the assistants installed.
var listModels = AvailableModels

// KnownModel reports whether model is valid for target, given the available ones.
func KnownModel(target, model string, available []string) bool {
	if t := targetByID(target); t != nil {
		return t.knownModel(model, available)
	}
	return false
}

// checkModels warns about models the assistant does not list; the install goes on.
func (r *run) checkModels() {
	for _, id := range r.opts.Targets {
		label := targetByID(id).label()
		available, ok := listModels(id)
		if !ok {
			r.log.Warn(label + ": could not list the available models (is a provider connected?); the chosen ones are not checked")
			continue
		}
		for _, p := range []string{"planner", "executor", "reviewer"} {
			if m := r.opts.Models[id][p]; !KnownModel(id, m, available) {
				r.log.Warn(fmt.Sprintf("%s: the %s model %q is not among the available models", label, p, m))
			}
		}
	}
}
