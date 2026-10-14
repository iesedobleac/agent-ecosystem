# agent-ecosystem

Installs a set of agents, skills, rules and MCP servers for Spec-Driven Development (SDD) and
Android work into the AI assistants you already use. One binary, an interactive menu, and the
same sources rendered natively for each assistant.

| Assistant | Status |
|---|---|
| OpenCode | Supported |
| Claude Code | Supported |
| Copilot CLI, Codex, Gemini CLI | Planned |

It never installs an assistant for you: it configures the ones you have.

## Install

**macOS (Homebrew)**

```bash
brew install --cask iesedobleac/tap/agent-ecosystem
```

**macOS / Linux (script)**: uses Homebrew on macOS when available, otherwise downloads the
binary from GitHub Releases and verifies its checksum.

```bash
curl -fsSL https://raw.githubusercontent.com/iesedobleac/agent-ecosystem/main/scripts/install.sh | bash
```

**Windows (PowerShell)**: installs from source with Go until the binaries are signed.

```powershell
irm https://raw.githubusercontent.com/iesedobleac/agent-ecosystem/main/scripts/install.ps1 | iex
```

**From source (any platform, Go 1.26+)**

```bash
go install github.com/iesedobleac/agent-ecosystem/cmd/agent-ecosystem@latest
```

Then run:

```bash
agent-ecosystem          # interactive installer
agent-ecosystem doctor   # read-only health check
```

### Requirements

Before writing anything, the installer lists the tools your selection needs and their state,
the way gentle-ai does:

| Tool | When it is needed | If it is missing |
|---|---|---|
| git | Always | The install stops |
| OpenCode | OpenCode selected | The install stops; install it from [opencode.ai](https://opencode.ai/docs) |
| Claude Code | Claude Code selected | Its configuration is prepared anyway |
| Node.js 18+ and npm | CodeGraph selected | Warning with the install command for your platform |
| Engram | Engram selected | Installed automatically (Homebrew, or its GitHub release) |
| CodeGraph | CodeGraph selected | Installed automatically with npm |
| Git Bash | Windows with OpenCode | Warning: the agents' shell commands need a POSIX shell |

It never installs Node.js or the assistants for you. Go is only needed to install from source
(Windows).

## What it installs

| Part | OpenCode (`~/.config/opencode`) | Claude Code (`~/.claude`) |
|---|---|---|
| Agents | 14 agents with per-agent permissions; orchestrators are primary agents | 14 subagents with tool lists; run an orchestrator with `claude --agent <name>` |
| Skills | `sdd-workflow`, `skill-generator` | Same |
| Rules | Managed block in `AGENTS.md` | Managed block in `CLAUDE.md` |
| Safety rules | Per-agent shell rules | Session deny rules: `git push`, `git reset --hard`, `git clean`, `git rebase` |
| Plugin | Context warning: toast past 200k tokens of context, then every +100k | Not needed |
| MCP servers | Engram and CodeGraph | Same |
| Memory | `engram setup opencode` | `engram setup claude-code` |

The agents:

- **SDD flow:** `sdd-orchestrator` with `sdd-discovery`, `sdd-constitution`, `sdd-spec`,
  `sdd-clarify`, `sdd-plan`, `sdd-tasks`, `sdd-apply`, `sdd-verify`.
- **Android:** `android-orchestrator` with `android-explore`, `android-worker`,
  `android-review`, `android-device` (these four also serve the SDD flow).

Outside OpenCode, permissions are coarser: each agent gets a list of tools, and command-level
limits come from the session deny rules and the agent's prompt.

## Command line

The menu covers everything; the same options work without it:

```bash
agent-ecosystem install --targets opencode,claude --dry-run
agent-ecosystem install --without android --claude-planner sonnet
agent-ecosystem sync          # re-apply the last selection, e.g. after upgrading the binary
agent-ecosystem uninstall     # remove everything it installed
```

| Flag | Effect |
|---|---|
| `--targets` | `opencode`, `claude` (default: the ones found on this machine) |
| `--without` | Skip components: `sdd`, `android`, `skills`, `plugin`, `engram`, `codegraph` |
| `--planner`, `--executor`, `--reviewer` | OpenCode models per profile |
| `--claude-planner`, `--claude-executor`, `--claude-reviewer` | Claude Code models per profile (default `opus`, `haiku`, `sonnet`) |
| `--config-dir`, `--claude-dir` | Assistant config directories |
| `--deliverables-dir`, `--android-sdk` | Paths the agents use |
| `--skip-deps`, `--no-shell-profile`, `--no-git-bash`, `--dry-run` | Behaviour |

Model profiles: **planner** for orchestrators, exploration and planning agents; **executor** for
`android-worker`, `android-device` and `sdd-apply`; **reviewer** for `android-review` and
`sdd-verify`.

## How it behaves on your machine

- **Safe re-runs.** A file you edited is copied to `~/.agent-ecosystem/backups/<date>/` before it
  is replaced.
- **Existing configuration is kept.** MCP servers with the same name are left untouched, rules go
  in a marked block, and deny rules are only added.
- **Comments in `opencode.jsonc` are lost.** The file is rewritten as plain JSON, after a backup.
- **State.** `~/.agent-ecosystem/state.json` records what was installed and the selection, which
  `sync` reuses and `uninstall` removes.

## Development

```bash
go test ./...
go run ./cmd/agent-ecosystem install --dry-run
```

`assets/` is the single source, embedded in the binary:

| Path | Contents |
|---|---|
| `assets/agents/<name>.json` | Description, mode, model profile, step limit and ordered permission rules |
| `assets/agents/<name>.md` | The prompt |
| `assets/skills/`, `assets/rules/rules.md`, `assets/mcp.json`, `assets/plugins/opencode/` | Skills, rules, MCP catalog, OpenCode plugins |

Prompts, skills and rules are templates:

- `{{read}}`, `{{grep}}`, `{{edit}}`, `{{webfetch}}` and similar tokens become each assistant's
  tool names.
- `{{#opencode}}…{{/opencode}}` and `{{#other}}…{{/other}}` hold text for OpenCode or for every
  other assistant.
- `@@PLACEHOLDERS@@` are machine paths resolved at install time.

`internal/render` defines the tokens and each assistant's file format.

Releases: push a `v*` tag. GoReleaser builds the macOS and Linux binaries, publishes them with
`checksums.txt`, and updates the cask in `iesedobleac/homebrew-tap` (secret
`HOMEBREW_TAP_GITHUB_TOKEN`).

## Known limitations

- Windows is not tested end to end. Agent prompts assume a POSIX shell, so on Windows the
  installer sets Git Bash as the OpenCode shell when it finds it.
- The Claude Code output follows Claude Code's documented format but has not been run there yet.
- With Claude models through GitHub Copilot in OpenCode, reaching a step limit fails with
  "does not support assistant message prefill", which is why only the reviewer agents have one.

## License

[MIT](LICENSE)
