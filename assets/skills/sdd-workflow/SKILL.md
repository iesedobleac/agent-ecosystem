---
name: sdd-workflow
description: Load the selected phase procedure before sdd-orchestrator coordinates SDD discovery, constitution, specs, requirement changes, clarification, planning, tasks, implementation, or verification. Not an executor skill.
---

# SDD Workflow

Use with `sdd-orchestrator`; this is a coordination playbook, not permission to
write, execute, or launch an agent. Keep the orchestrator's permanent safeguards.

## Select One Procedure

Read the matching reference relative to this skill's base directory BEFORE
phase-specific checks, interviews, or delegation. Do not read all references.

| Requested work | Reference | Delegate | Entry condition |
|---|---|---|---|
| Project discovery / AGENTS.md | [discovery](references/discovery.md) | sdd-discovery | Check existing project instructions first |
| Constitution | [constitution](references/constitution.md) | sdd-constitution | User requested it; location check first |
| New functional spec | [specification](references/specification.md) | sdd-spec | Draft only until content-and-save approval |
| Existing-spec requirement change | [requirement-change](references/requirement-change.md) | sdd-spec | Diff proposal first; exact baseline |
| Spec ambiguity review | [clarification](references/clarification.md) | sdd-clarify | Explicit request; findings only |
| Technical plan | [planning](references/planning.md) | sdd-plan | Specific spec; draft only |
| Task decomposition | [tasks](references/tasks.md) | sdd-tasks | Approved plan; draft only |
| Implement one task | [implementation](references/implementation.md) | sdd-apply | Explicit task execution authority; current docs |
| RF compliance review | [verification](references/verification.md) | sdd-verify | Explicit review authority and full/partial scope |

Ambiguous requests need clarification, not speculative loading or execution.
For simple file-location work use `android-explore` under existing permissions; do not
invent a feature or interview. Each procedure contains its own entry checks.

## Load, Delegate, Stop

- Reuse an already available, current procedure; do not reread it each turn.
  After compaction or missing context, reload this skill and the needed reference
  before acting. If either is unavailable, report the blocker; do not improvise.
- Supply objective, root/source paths and versions, relevant instructions and
  constitution, scope, decisions, ledger, approval state, testing mode, agreed
  verification, and expected output. Children have fresh context. Pass applicable
  constraints, not this whole playbook or unrelated phase procedures.
- Preserve policy, question count, exact approvals, baseline, source linkage,
  and downstream reconciliation state across sessions. Loaded instructions do
  not supply missing decisions or approval history.
- A recommendation/link to another phase is not authorization to start it or
  read its procedure. Load another reference only when the requested work needs
  it; perform that phase only after its own authorization and entry checks.
- Review real outputs against the selected procedure and stop at its gate.
  Keep drafting/reports out of destination files until the appropriate saving
  approval. Plan and tasks drafts live in the {{tmp_label}} temp `sdd-drafts/` folder
  (see the orchestrator's Drafts section); other drafts stay in the conversation.

This skill does not hide or unload content already in a session. Its purpose is
to avoid loading unrelated procedures, not to guarantee a token/cost percentage.
