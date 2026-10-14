## Shared Rules

- Never add "Co-Authored-By" or AI attribution to commits. Use Conventional Commits.
- Generated artifacts (code, comments, UI copy, docs, specs, tests, commit
  messages, memory entries) default to English unless the user requests another
  language or the project already uses one for that artifact. When Spanish is
  used, it is neutral and professional. Never put persona voice, slang, or
  regional phrasing into any artifact.

## CodeGraph

Use CodeGraph before broad filesystem searches for structural or codebase
questions: repository maps, architecture, call flow, dependencies, symbol
references, impact analysis, and "how does X work" questions.

### Querying

- Call CodeGraph's `codegraph_explore` tool first, before {{grep}}, {{glob}}, or
  shell searches. Pass the repository root path as
  `projectPath`; CodeGraph resolves the nearest `.codegraph/` index at or above
  that path.
{{#opencode}}- How to call MCP tools: with Code Mode (OpenCode 2 default) use the `execute`
  tool, for example
  `return await tools.codegraph.codegraph_explore({ query, projectPath })`, and
  Engram as `tools.engram.mem_search(...)`. When a tool is listed directly (for
  example `codegraph_codegraph_explore`), call it directly.
{{/opencode}}{{#other}}- MCP tools are named after their server, for example {{mcp_example}}.
{{/other}}- Treat source returned by `codegraph_explore` as already read; do not re-read
  the same files.
- If the MCP tool is unavailable, use the read-only upstream CLI commands:
  `codegraph status`, `query`, `explore`, `node`, `files`, `callers`,
  `callees`, `impact`, and `affected`.
- Never query another workspace's or another checkout's `.codegraph/` index.

### Index lifecycle (not owned by agents)

- Index creation and refresh belong to the developer: `codegraph init` in the
  repository root creates the index.
- Agents never run `codegraph init`, `index`, `uninit`, `install`,
  `uninstall`, `upgrade`, or `gentle-ai codegraph init`.
- The CodeGraph file watcher keeps indexes in sync after edits. Do not run
  `codegraph sync` routinely.
- If a repository has no `.codegraph/` index or CodeGraph fails, say so
  briefly, suggest running `codegraph init` in the repository root, and fall back to
  `read`, `grep`, and `glob`.

### Worktrees

- Create Git worktrees that need CodeGraph under the user's home directory,
  never under `/tmp` or `/var/tmp`.
- Every worktree needs its own `.codegraph/` index. Never copy, symlink, or
  reuse another checkout's index.
