package installer

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// nodeHint suggests how to install Node.js on this platform.
func nodeHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install node"
	case "windows":
		return "winget install OpenJS.NodeJS.LTS"
	}
	switch {
	case which("apt-get") != "":
		return "NodeSource LTS setup, then: sudo apt-get install -y nodejs"
	case which("pacman") != "":
		return "sudo pacman -S --noconfirm nodejs npm"
	case which("dnf") != "":
		return "NodeSource LTS setup, then: sudo dnf install -y nodejs"
	}
	return "install Node.js LTS from https://nodejs.org"
}

func gitHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "xcode-select --install (or brew install git)"
	case "windows":
		return "winget install Git.Git"
	}
	switch {
	case which("apt-get") != "":
		return "sudo apt-get install -y git"
	case which("pacman") != "":
		return "sudo pacman -S --noconfirm git"
	case which("dnf") != "":
		return "sudo dnf install -y git"
	}
	return "install git from https://git-scm.com"
}

// Status of a required tool.
type Status int

const (
	// Present: the tool is available.
	Present Status = iota
	// WillInstall: missing, and the installer installs it.
	WillInstall
	// Optional: missing; the install continues without it.
	Optional
	// Blocking: missing; the install cannot continue.
	Blocking
)

// Requirement is one tool the selection needs.
type Requirement struct {
	Name    string
	Purpose string
	Status  Status
	Detail  string // version, path or how it is installed
	Hint    string // how to install it when missing
}

var majorRe = regexp.MustCompile(`(\d+)\.`)

func version(bin string, args ...string) (string, bool) {
	out, err := command(bin, args...)
	if err != nil {
		return "", false
	}
	return lastLine(out), true
}

// Requirements lists the tools the selection needs and their state, the way
// gentle-ai does before installing: what is present, what the installer adds,
// what is missing but optional and what blocks the install.
func Requirements(opts Options) []Requirement {
	c := opts.Components
	reqs := []Requirement{{Name: "git", Purpose: "required by the installer and the agents"}}
	if v, ok := version("git", "--version"); ok {
		reqs[0].Detail = strings.TrimPrefix(v, "git version ")
	} else {
		reqs[0].Status, reqs[0].Hint = Blocking, gitHint()
	}

	npm := which("npm") != ""
	if c.CodeGraph {
		node := Requirement{Name: "Node.js 18+ and npm", Purpose: "CodeGraph runs on Node.js"}
		v, ok := version("node", "--version")
		switch {
		case !ok || !npm:
			node.Status, node.Hint = Optional, nodeHint()
		default:
			node.Detail = v
			if m := majorRe.FindStringSubmatch(strings.TrimPrefix(v, "v")); m != nil {
				if major, _ := strconv.Atoi(m[1]); major < 18 {
					node.Status, node.Hint = Optional, "Node.js "+v+" is too old: "+nodeHint()
				}
			}
		}
		reqs = append(reqs, node)
	}

	if c.Engram {
		r := Requirement{Name: "Engram", Purpose: "persistent memory for the agents"}
		switch {
		case which("engram") != "":
			r.Detail, _ = version("engram", "version")
		case opts.SkipDeps:
			r.Status, r.Hint = Optional, "brew install gentleman-programming/tap/engram (skipped with --skip-deps)"
		case runtime.GOOS != "windows" && which("brew") != "":
			r.Status, r.Detail = WillInstall, "with Homebrew (gentleman-programming/tap/engram)"
		default:
			r.Status, r.Detail = WillInstall, "from its GitHub release"
		}
		reqs = append(reqs, r)
	}

	if c.CodeGraph {
		r := Requirement{Name: "CodeGraph", Purpose: "code graph the agents query before reading files"}
		switch {
		case which("codegraph") != "":
			r.Detail = which("codegraph")
		case opts.SkipDeps:
			r.Status, r.Hint = Optional, "npm install -g @colbymchenry/codegraph (skipped with --skip-deps)"
		case npm:
			r.Status, r.Detail = WillInstall, "with npm (@colbymchenry/codegraph)"
		default:
			r.Status, r.Hint = Optional, "needs Node.js and npm first"
		}
		reqs = append(reqs, r)
	}

	for _, t := range opts.Targets {
		switch t {
		case "opencode":
			r := Requirement{Name: "OpenCode", Purpose: "selected assistant"}
			if v, ok := version("opencode", "--version"); ok {
				r.Detail = v
			} else {
				r.Status, r.Hint = Blocking, "install it from https://opencode.ai/docs and run again"
			}
			reqs = append(reqs, r)
			if runtime.GOOS == "windows" && !opts.NoGitBash {
				gb := Requirement{Name: "Git Bash", Purpose: "the agents' shell commands need a POSIX shell"}
				if p := gitBash(); p != "" {
					gb.Detail = p
				} else {
					gb.Status, gb.Hint = Optional, "winget install Git.Git"
				}
				reqs = append(reqs, gb)
			}
		case "claude":
			r := Requirement{Name: "Claude Code", Purpose: "selected assistant"}
			if v, ok := version("claude", "--version"); ok {
				r.Detail = v
			} else {
				r.Status, r.Hint = Optional, "its configuration is prepared anyway; install it from https://docs.claude.com/en/docs/claude-code/setup"
			}
			reqs = append(reqs, r)
		}
	}
	return reqs
}

// Blockers returns the requirements that stop the install.
func Blockers(reqs []Requirement) []Requirement {
	var out []Requirement
	for _, r := range reqs {
		if r.Status == Blocking {
			out = append(out, r)
		}
	}
	return out
}

func gitBash() string {
	for _, bash := range []string{"C:/Program Files/Git/bin/bash.exe", "C:/Program Files (x86)/Git/bin/bash.exe"} {
		if _, err := os.Stat(bash); err == nil {
			return bash
		}
	}
	return ""
}

// checkRequirements is replaced in tests, which run without the assistants installed.
var checkRequirements = Requirements

func (r *run) ensureDependencies(targets []string) error {
	if which("git") == "" {
		return fmt.Errorf("git is required")
	}
	for _, name := range []string{"opencode", "codegraph", "engram", "claude"} {
		r.bins[name] = which(name)
	}
	if r.opts.SkipDeps {
		return nil
	}
	c := r.opts.Components
	if which("node") == "" || which("npm") == "" {
		r.log.Warn("Node.js/npm not found: " + nodeHint())
	}
	if c.CodeGraph && r.bins["codegraph"] == "" {
		if which("npm") == "" {
			r.log.Warn("codegraph needs npm; install Node.js and run again")
		} else if r.opts.DryRun {
			r.log.Info("[dry-run] npm install -g @colbymchenry/codegraph")
		} else {
			r.log.Info("Installing CodeGraph (npm @colbymchenry/codegraph)")
			if out, err := command("npm", "install", "-g", "@colbymchenry/codegraph"); err != nil {
				r.log.Warn("npm install failed: " + lastLine(out))
			}
			r.bins["codegraph"] = which("codegraph")
		}
	}
	if c.Engram && r.bins["engram"] == "" {
		if r.opts.DryRun {
			r.log.Info("[dry-run] install engram")
		} else if p, err := r.installEngram(); err != nil {
			r.log.Warn("engram: " + err.Error())
		} else {
			r.bins["engram"] = p
		}
	}
	for _, name := range []string{"codegraph", "engram"} {
		if (name == "codegraph" && !c.CodeGraph) || (name == "engram" && !c.Engram) {
			continue
		}
		if r.bins[name] != "" {
			r.log.OK(name + ": " + r.bins[name])
		} else if !r.opts.DryRun {
			r.log.Warn(name + " is not available; install it manually")
		}
	}
	for _, t := range targets {
		bin := map[string]string{"opencode": "opencode", "claude": "claude"}[t]
		if r.bins[bin] == "" {
			r.log.Warn(TargetLabels[t] + " not found: its configuration is prepared, install it to use it")
		}
	}
	return nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}

func (r *run) installEngram() (string, error) {
	if runtime.GOOS != "windows" && which("brew") != "" {
		r.log.Info("Installing engram with Homebrew")
		if _, err := command("brew", "help", "trust"); err == nil {
			_, _ = command("brew", "trust", "--formula", "gentleman-programming/tap/engram")
		}
		if out, err := command("brew", "install", "gentleman-programming/tap/engram"); err == nil {
			return which("engram"), nil
		} else {
			r.log.Warn("brew install engram failed: " + lastLine(out))
		}
	}
	r.log.Info("Downloading engram from GitHub releases")
	arch := map[string]string{"amd64": "amd64", "arm64": "arm64"}[runtime.GOARCH]
	if arch == "" {
		return "", fmt.Errorf("no engram binary for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get("https://api.github.com/repos/Gentleman-Programming/engram/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var release struct {
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	suffix := fmt.Sprintf("_%s_%s.", runtime.GOOS, arch)
	var url, name string
	for _, a := range release.Assets {
		if strings.Contains(a.Name, suffix) {
			url, name = a.URL, a.Name
		}
	}
	if url == "" {
		return "", fmt.Errorf("release asset for %s/%s not found", runtime.GOOS, arch)
	}
	archive, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer archive.Body.Close()
	data, err := io.ReadAll(archive.Body)
	if err != nil {
		return "", err
	}
	exe := "engram"
	binDir := filepath.Join(r.home, ".local", "bin")
	if runtime.GOOS == "windows" {
		exe, binDir = "engram.exe", filepath.Join(r.home, "bin")
	}
	binary, err := extract(name, data, exe)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(binDir, exe)
	if err := os.WriteFile(target, binary, 0o755); err != nil {
		return "", err
	}
	if !strings.Contains(string(os.PathListSeparator)+os.Getenv("PATH")+string(os.PathListSeparator), string(os.PathListSeparator)+binDir+string(os.PathListSeparator)) {
		r.log.Warn("Add " + binDir + " to your PATH")
	}
	return slash(target), nil
}

// extract returns the named file from a .tar.gz or .zip archive.
func extract(archiveName string, data []byte, file string) ([]byte, error) {
	if strings.HasSuffix(archiveName, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if filepath.Base(f.Name) == file {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, fmt.Errorf("%s not found in %s", file, archiveName)
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%s not found in %s", file, archiveName)
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(h.Name) == file {
			return io.ReadAll(tr)
		}
	}
}

// runEngramSetup runs `engram setup <agent>`, optionally with extra environment.
func (r *run) runEngramSetup(agent string, env ...string) {
	if r.opts.SkipDeps || r.opts.DryRun || !r.opts.Components.Engram || r.bins["engram"] == "" {
		return
	}
	r.log.Info("Configuring the Engram plugin (engram setup " + agent + ")")
	cmd := exec.Command(r.bins["engram"], "setup", agent)
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		r.log.Warn("engram setup " + agent + " failed: " + lastLine(string(out)))
	}
}
