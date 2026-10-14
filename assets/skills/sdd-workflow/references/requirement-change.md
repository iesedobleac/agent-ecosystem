# Existing-Spec Requirement Changes

Use when the user requests adding, modifying, or retiring a requirement in an
existing spec. "Update the spec first and show the diff" means propose a diff
first, not save before approval. Do not create another subagent for this work.

1. Identify the affected spec; ask if ambiguous before delegating. Establish root,
   exact baseline/version, local work, new requirement and scope, constitution,
   confirmed decisions, feature policy, and existing question ledger. Reuse the
   policy without another testing question. Inspect related plan/tasks for impact.
2. Delegate revision analysis to `sdd-spec` with that context and explicitly NO
   writing, code, technical decisions, or derived-document edits. Require the full
   revised draft, proposed diff, RF/criterion changes, sourced impacts, and blockers.
3. Relay necessary functional questions one at a time under the existing total
   six-question limit. Preserve the ledger across sessions; do not reset it for
   a change request or fill missing history with assumptions. Stop on unresolved
   consequential gaps, including uncertainty about IDs or source versions.
4. Review the complete proposal and diff: no unrelated changes, stable surviving
   RF IDs, unused new IDs, no reuse of known retired IDs, explicit authorized
   retirements, EARS, scope/exclusions and completion criteria consistent with the
   new requirement and constitution. Separate functional changes from potential
   technical impact; do not invent a design while revising the spec.
5. Present the proposed diff, summary, and downstream impact to the user. Obtain
   explicit approval of BOTH revised content and saving the same spec. A correction
   or initial change request does not authorize writing. If approval applies only
   to part of the proposal, reconcile the complete draft before delegating writes.
6. Relay exact approved content, destination and reference baseline to `sdd-spec`
   for that file only. Require a recheck for concurrent changes before writing.
   Any changed baseline blocks saving; do not overwrite or silently merge. Read
   the result and compare it with the approved revision/diff.
7. Report the saved spec and identified affected plan sections/tasks, distinguishing
   confirmed impact from uncertainty. Stop and ask whether the user wants those
   documents reviewed. Do not automatically alter plan/tasks, reset progress,
   implement code, or invoke clarify/verify. Missing downstream docs are reported,
   not automatically created.
8. After separate authorization, use `sdd-plan` to revise the technical plan and
   obtain its content-and-save approval. Then use `sdd-tasks` for affected task
   revisions with their own approval. Preserve IDs, recorded work and checkboxes;
   never erase progress silently. Execution still needs separate authorization.
   Load `planning.md`, `tasks.md`, or `implementation.md` only when that phase
   becomes authorized and needed; this step is not advance permission to run them.

Maintain the revision baseline, decisions, and downstream reconciliation state
in the feature context passed to resumed sessions. Do not launch affected tasks
while plan/tasks reconciliation is pending. A document's existence does not prove
that its approval applies to the current requirements; if linkage cannot be
established, report uncertainty and stop affected execution. Unaffected tasks may
continue only with justified independence and explicit execution authorization.
Do not perform mass rewrites or create extra change/impact files. Global agent
configuration changes do not themselves authorize revising a project's documents.
