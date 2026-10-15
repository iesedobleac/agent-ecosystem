package installer

import (
	"fmt"

	"github.com/iesedobleac/iesedoble-ai/internal/render"
)

// AvailableModels lists the models a target can use, or says why it cannot:
// OpenCode missing, failing, timing out or without a connected provider.
func AvailableModels(target string) ([]string, error) {
	if t := targetByID(target); t != nil {
		return t.availableModels()
	}
	return nil, fmt.Errorf("unknown target %q", target)
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
		available, err := listModels(id)
		if err != nil {
			r.log.Warn(label + ": could not list the available models, so the chosen ones are not checked (" + err.Error() + ")")
			continue
		}
		for _, p := range render.Profiles {
			if m := r.opts.Models[id][p]; !KnownModel(id, m, available) {
				r.log.Warn(fmt.Sprintf("%s: the %s model %q is not among the available models", label, p, m))
			}
		}
	}
}
