package installer

import (
	"os"
	"path/filepath"

	"github.com/iesedobleac/iesedoble-ai/internal/render"
)

// Target is one assistant the ecosystem installs into. Install, Uninstall,
// Doctor and the requirement checks go through it instead of switching on the
// assistant: adding one means a new file and an entry in targets.
type Target interface {
	id() string    // key in options, state and template blocks ({{#opencode}})
	label() string // human name
	// flagPrefix goes before the model flags: --<prefix>planner.
	flagPrefix() string
	binary() string // command that runs the assistant
	defaultModels() render.Models
	// defaultEfforts are the reasoning effort per profile; nil when the
	// assistant has no per-agent effort.
	defaultEfforts() render.Efforts
	// availableModels lists the models it can use, or says why it cannot.
	availableModels() ([]string, error)
	// knownModel reports whether model is valid, given the listed ones.
	knownModel(model string, available []string) bool
	// detected reports whether the assistant or its config is on this machine.
	detected(r *run) bool
	// requirements are the tools this assistant needs, after the shared ones.
	requirements(opts Options) []Requirement
	// dirs are the paths its rendered files refer to.
	dirs(r *run) targetDirs
	// steps is how many progress steps install reports.
	steps() int
	install(r *run, src *sources, st *TargetState) error
	uninstall(r *run, st *TargetState) error
	nextStep() string
}

type targetDirs struct {
	tmp, data, skills string
}

// targets are the supported assistants, in display order.
var targets = []Target{openCode{}, claudeCode{}}

func targetByID(id string) Target {
	for _, t := range targets {
		if t.id() == id {
			return t
		}
	}
	return nil
}

// AllTargets are the supported assistants' IDs, in display order.
var AllTargets = func() []string {
	ids := make([]string, len(targets))
	for i, t := range targets {
		ids[i] = t.id()
	}
	return ids
}()

// TargetLabels are human names for the targets.
var TargetLabels = func() map[string]string {
	m := map[string]string{}
	for _, t := range targets {
		m[t.id()] = t.label()
	}
	return m
}()

// DefaultModels per target and profile.
var DefaultModels = func() map[string]render.Models {
	m := map[string]render.Models{}
	for _, t := range targets {
		m[t.id()] = t.defaultModels()
	}
	return m
}()

// DefaultEfforts per target and profile, for the targets with per-agent effort.
var DefaultEfforts = func() map[string]render.Efforts {
	m := map[string]render.Efforts{}
	for _, t := range targets {
		if e := t.defaultEfforts(); e != nil {
			m[t.id()] = e
		}
	}
	return m
}()

// EffortLevels are the effort choices; "auto" keeps the session's level.
var EffortLevels = []string{"auto", "low", "medium", "high", "xhigh", "max"}

// AutoCompactWindows are the Claude Code auto-compact windows offered, in
// tokens; 0 keeps Claude Code's default. Any value in range works from the CLI.
var AutoCompactWindows = []int{0, 100_000, 150_000, 200_000, 300_000, 500_000}

// Auto-compact window range accepted by Claude Code.
const (
	MinAutoCompact = 100_000
	MaxAutoCompact = 1_000_000
)

// ModelFlagPrefix is what goes before a target's model flags (--<prefix>planner).
func ModelFlagPrefix(id string) string {
	if t := targetByID(id); t != nil {
		return t.flagPrefix()
	}
	return id + "-"
}

// tempDir creates and returns the temp folder the agents use, preferring the
// assistant's own when it reports one.
func tempDir(preferred, name string) string {
	dir := preferred
	if dir == "" {
		dir = filepath.Join(os.TempDir(), name)
	}
	_ = os.MkdirAll(dir, 0o755)
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	return dir
}
