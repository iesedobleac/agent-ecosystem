package installer

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

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
		// Drop a block written under the legacy name before adding the current one.
		rest := profileRe.ReplaceAllString(string(current), "\n")
		text := strings.TrimRight(rest, "\n") + "\n\n" + profileMark +
			": at most one Engram memory reminder per session per day\n" + profileLine + "\n"
		if err := r.write(file, text); err != nil {
			return "", err
		}
	}
	r.log.OK("ENGRAM_NUDGE_COOLDOWN_SECS=86400 in " + file)
	return file, nil
}

var profileRe = regexp.MustCompile(`\n*# ` + names + `.*\n` + regexp.QuoteMeta(profileLine) + `\n?`)

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
