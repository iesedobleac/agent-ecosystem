# Discovery for Existing Projects

Use this workflow when the user asks to understand or document a project, prepare
an AGENTS.md, or adopt SDD in an existing project. A full interview is not needed
for every small change.

Before invoking `sdd-discovery`, use read-only tools to check whether the project
already has an AGENTS.md at its root or project-owned AGENTS.md instructions
applicable to the subproject you will work on. Do not confuse {{product}}'s global
AGENTS.md with the project's own documentation.

- If one exists, read it and use it as context. Do not invoke `sdd-discovery` or
  start a discovery interview by default, even if it is brief or incomplete.
  Resolve task-specific gaps through focused exploration or concrete questions.
- In that case, you may invoke `sdd-discovery` only if the user explicitly asks
  to review, update, or rewrite that AGENTS.md, or explicitly requests the
  subagent. A general request to understand the project does not authorize
  repeating discovery.
- If no applicable project-owned AGENTS.md exists, you may proceed with this
  workflow.
- If you cannot check whether it exists, report the blocker before delegating;
  do not assume it is missing.

1. Delegate exploration to `sdd-discovery`, providing the project path, objective,
   existing instructions, and any confirmed context.
2. Review its findings and ask the user relevant questions in small batches
   (up to three per round). Keep the interview in the main conversation; do not
   make the subagent depend on direct answers from the user.
3. Continue the subagent session with the answers if the tool supports it. If a
   new session starts, also provide findings, sources, decisions, and open
   questions. Do not repeat questions that have already been answered.
4. Stop the interview once there is enough context to work safely. Present the
   concise AGENTS.md draft, uncertainties, and pending checks. Compare proposed
   instructions against existing ones.
5. Only after explicit approval of both the content and saving it, delegate
   writing to `android-worker` at the agreed path, preserving existing work. Then review
   the resulting file. `sdd-discovery` has no editing permissions.
