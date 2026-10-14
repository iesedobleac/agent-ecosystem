# Install and usage guide

← [Back to README](../README.md)

This guide covers macOS (Apple Silicon and Intel) and Windows. Commands are the same on both unless
a section says otherwise; on Windows, run them in PowerShell.

- [Before you start](#before-you-start)
- [Install](#install)
- [First run](#first-run)
- [Use the agents](#use-the-agents)
- [Keep it up to date](#keep-it-up-to-date)
- [Check your install](#check-your-install)
- [Uninstall](#uninstall)
- [Troubleshooting](#troubleshooting)

---

## Before you start

### macOS

You need:

- **git**. If it is missing, install the Xcode command line tools with `xcode-select --install`,
  or run `brew install git`.
- **At least one assistant**: [OpenCode](https://opencode.ai/docs) or
  [Claude Code](https://docs.claude.com/en/docs/claude-code/setup). The installer configures the
  assistants you have; it never installs them.
- **Node.js 18+ and npm**, only if you want CodeGraph: `brew install node`.

[Homebrew](https://brew.sh) is recommended. With it, the installer also installs Engram for you.

### Windows

Windows installs from source, so you need Go:

```powershell
winget install Git.Git
winget install GoLang.Go
winget install OpenJS.NodeJS.LTS   # only if you want CodeGraph
```

Git for Windows also brings Git Bash, which the agents' shell commands need. Install at least one
assistant too: [OpenCode](https://opencode.ai/docs) or
[Claude Code](https://docs.claude.com/en/docs/claude-code/setup).

Open a new terminal afterwards, so the new `PATH` is loaded.

## Install

### macOS: Homebrew (recommended)

```bash
brew install --cask iesedobleac/tap/agent-ecosystem
```

The binary is not notarized by Apple; the cask removes the quarantine flag on install, so macOS
runs it without a warning.

### macOS: install script

```bash
curl -fsSL https://raw.githubusercontent.com/iesedobleac/agent-ecosystem/main/scripts/install.sh | bash
```

On macOS with Homebrew the script installs the cask above. To install the binary instead,
downloaded from GitHub Releases and checked against `checksums.txt`:

```bash
curl -fsSL https://raw.githubusercontent.com/iesedobleac/agent-ecosystem/main/scripts/install.sh | bash -s -- --method binary
```

The binary goes to `/usr/local/bin` when it is writable, otherwise to `~/.local/bin`; choose
another folder with `--dir <folder>`. If that folder is not in your `PATH`, the script tells you
the line to add.

### Windows: install script

```powershell
irm https://raw.githubusercontent.com/iesedobleac/agent-ecosystem/main/scripts/install.ps1 | iex
```

The script checks git and Go, then builds and installs the binary with `go install`. If Go's
binary folder (`%USERPROFILE%\go\bin`) is not in your `PATH`, it prints the command to add it;
run it and open a new terminal.

### From source

With Go 1.26 or later, on either system:

```bash
go install github.com/iesedobleac/agent-ecosystem/cmd/agent-ecosystem@latest
```

Check the install:

```bash
agent-ecosystem version
```

## First run

```bash
agent-ecosystem
```

Choose **Install or update** and go through the four steps. Move with `↑`/`↓`, toggle with
`space`, continue with `enter`, go back with `esc`.

1. **Assistants.** The ones found on this machine are marked as detected and preselected.
2. **Components.** All but the subagent monitor are selected by default:

   | Component | What it adds |
   |---|---|
   | SDD flow | `sdd-orchestrator`, its phase agents and the `sdd-workflow` skill |
   | Android orchestrator | `android-orchestrator` (its subagents also come with the SDD flow) |
   | General skills | `skill-generator` |
   | Context warning | OpenCode toast when a session passes 200k tokens |
   | Engram memory | Persistent memory MCP server and plugin |
   | CodeGraph | Code graph MCP server |
   | Subagent monitor | Keeps the OpenCode subagent statusline that Engram's setup adds. Off by default: the installer removes it from `tui.json` and `cli.json`, after a backup |

3. **Models.** One model per role and assistant: **planner** (orchestrators and planning),
   **executor** (implementation and device work) and **reviewer** (review and verification).
   Press `enter` on a row to pick from the models the assistant offers, and type to filter.
   OpenCode lists the models of the providers you have connected; Claude Code offers `opus`,
   `sonnet`, `haiku`, `fable` and `inherit`. **Custom…** accepts any other model ID.
4. **Review.** Check the summary and choose **Install**, or **Dry run** to see what would change
   without writing anything.

Before writing, the installer shows the tools your selection needs. `✓` is present, `↓` will be
installed for you (Engram and CodeGraph), `!` is missing but optional, and `✗` stops the install
until you install it.

At the end you get the next steps for each assistant.

On Windows, Engram is downloaded from its GitHub release to `%USERPROFILE%\bin`. If the installer
warns that this folder is not in your `PATH`, add it and open a new terminal:

```powershell
[Environment]::SetEnvironmentVariable('Path', $env:Path + ";$env:USERPROFILE\bin", 'User')
```

### Without the menu

The same install, from the command line:

```bash
agent-ecosystem install                       # detected assistants, everything, default models
agent-ecosystem install --dry-run             # show what would change
agent-ecosystem install --targets claude --without android,codegraph
agent-ecosystem install --with subagent-monitor
agent-ecosystem install --planner openai/gpt-5.5 --claude-reviewer opus
```

Run `agent-ecosystem install --help` for every flag. A model the assistant does not list produces
a warning and the install goes on.

## Use the agents

### OpenCode

1. Restart OpenCode. If it runs as a background service, also run `opencode service restart` from
   a new terminal.
2. Connect your model provider with `/connect`, if you have not yet.
3. Pick `sdd-orchestrator` or `android-orchestrator` as the agent and describe what you need.

### Claude Code

Start a session with an orchestrator as the main agent:

```bash
claude --agent sdd-orchestrator
claude --agent android-orchestrator
```

The other twelve agents are subagents: the orchestrators delegate to them.

### CodeGraph

Index each repository once, from its root:

```bash
codegraph init
```

### Which orchestrator

- **`sdd-orchestrator`** for Spec-Driven Development: project discovery, a constitution,
  functional specs, a technical plan, tasks, implementation one task at a time, and verification.
  Each phase runs when you ask for it, and nothing is saved without your approval.
- **`android-orchestrator`** for small and medium Android tasks without SDD phases.

## Keep it up to date

macOS:

```bash
brew upgrade --cask iesedobleac/tap/agent-ecosystem
agent-ecosystem sync
```

Windows:

```powershell
go install github.com/iesedobleac/agent-ecosystem/cmd/agent-ecosystem@latest
agent-ecosystem sync
```

`sync` re-applies your last selection with the new version. Files you edited are backed up
before they are replaced.

On macOS, if you installed with the script's binary method, run the script again instead of
`brew upgrade`. Restart OpenCode afterwards (`opencode service restart`).

## Check your install

On macOS:

```bash
agent-ecosystem doctor
```

It only reads: the tools, the recorded install, and whether installed files are missing or were
edited by hand.

## Uninstall

On macOS:

```bash
agent-ecosystem uninstall
brew uninstall --cask agent-ecosystem
```

`uninstall` removes what it added: agents, skills, the rules block in `AGENTS.md` / `CLAUDE.md`,
its MCP servers and the Claude Code deny rules, and with the last assistant also the line it added
to your shell profile. Engram, CodeGraph and the assistants stay installed. To remove a single
assistant, pass `--targets opencode` or `--targets claude`; add `--dry-run` to see what would be
removed.

## Troubleshooting

**The models step shows "choose a model" for OpenCode, or no models at all.**
OpenCode only lists the models of connected providers. Connect one in OpenCode with `/connect`
and run `agent-ecosystem` again, or use **Custom…** and type the model ID. When the models cannot
be listed, the menu and `agent-ecosystem install` show why, with OpenCode's own error.

**An OpenCode subagent statusline still shows up.**
Engram's setup adds it; the installer removes it unless **Subagent monitor** is selected. Run
`agent-ecosystem sync`, then `opencode service restart`, and check with `opencode plugin list`.

**The install stops at "Required tools".**
A tool marked `✗` is missing: install it with the hint shown next to it and run again. **Dry run**
still works, to see what would change.

**A file I changed was replaced.**
It was backed up first. Backups are in `~/.agent-ecosystem/backups/<date>/`, with the same path
as the original.

**My comments in `opencode.jsonc` are gone.**
The installer rewrites that file as plain JSON. The original, with comments, is in the backups
folder.

**Where things are** (on Windows, `~` is your user folder, `%USERPROFILE%`)

| What | Where |
|---|---|
| Install record and last selection | `~/.agent-ecosystem/state.json` |
| Backups | `~/.agent-ecosystem/backups/` |
| OpenCode files | `~/.config/opencode` (or `--config-dir`) |
| Claude Code files | `~/.claude`, or `$CLAUDE_CONFIG_DIR` (or `--claude-dir`) |
| Agent deliverables | `~/agent-deliverables` (or `--deliverables-dir`) |
