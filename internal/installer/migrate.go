package installer

import (
	"os"
	"path/filepath"
	"regexp"
)

// legacyName is the tool's name before v0.4.0. Installs made with it keep their
// state in ~/.agent-ecosystem and mark their blocks with it; they are migrated
// on the next install, sync or uninstall.
const legacyName = "agent-ecosystem"

// names matches the marker name of current and legacy installs.
var names = `(?:` + regexp.QuoteMeta(toolName) + `|` + regexp.QuoteMeta(legacyName) + `)`

// stateDirFor is where the state lives: ~/.iesedoble-ai, or the legacy folder
// while an older install has not been migrated yet.
func stateDirFor(home string) string {
	dir := filepath.Join(home, "."+toolName)
	if _, err := os.Stat(dir); err != nil {
		legacy := filepath.Join(home, "."+legacyName)
		if _, err := os.Stat(legacy); err == nil {
			return legacy
		}
	}
	return dir
}

// migrateStateDir moves the legacy state folder to the current one, unless
// this is a dry run or both exist.
func migrateStateDir(opts Options, log Logger) {
	if opts.DryRun {
		return
	}
	home := opts.Home
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	dir, legacy := filepath.Join(home, "."+toolName), filepath.Join(home, "."+legacyName)
	if _, err := os.Stat(dir); err == nil {
		return
	}
	if _, err := os.Stat(legacy); err != nil {
		return
	}
	if err := os.Rename(legacy, dir); err != nil {
		log.Warn("Could not move " + legacy + " to " + dir + ": " + err.Error())
		return
	}
	log.Info("Moved " + legacy + " to " + dir)
}
