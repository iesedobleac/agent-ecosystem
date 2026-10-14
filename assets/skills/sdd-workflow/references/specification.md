# Functional Specifications

Use this workflow when the user requests a functional specification. Do not
start a new feature specification merely because a small change was requested.
When the request changes requirements in an existing spec, use
`requirement-change.md` instead of recreating the feature or resetting its RFs.

1. Establish the project root. Read applicable AGENTS.md, the canonical
   `<project-root>/docs/constitution.md`, and supplied documentation. If required
   documents are missing, unreadable, or conflicting, clarify the blocker; do not
   silently create documents or change principles. Honor the location rules in
   `constitution.md` when constitution work is requested; do not start that work
   or load its procedure merely because a spec needs constitutional context.
2. Agree a safe, single-segment specification folder name with the user, using
   `<jira-key-lowercase>-<kebab-slug>`. The only destination is
   `<project-root>/ai/specs/<spec-name>/spec.md`. Check for existing work and obtain
   explicit authorization before replacing an existing spec.
3. Delegate to `sdd-spec` with the root, objective, sources, relevant instructions,
   constitution, confirmed context, scope, destination, existing spec if any, and
   expected output. Initially authorize analysis and drafting only, not editing.
4. Relay exactly one clarification question at a time and wait for its answer.
   This workflow has a maximum of six questions per specification, not per child
   session. Questions concern scope, edge cases, and errors; do not bundle them
   or apply the three-question batches used by discovery and constitution agents.
   Keep a cumulative numbered question-and-answer ledger. Continue the same child
   session when possible; otherwise supply the entire ledger, count, decisions,
   sources, and findings. Stop early when sufficient context exists. If important
   gaps remain after six questions, report the blocker without a seventh question
   or invented behavior. Folder confirmation and approval are administrative
   steps, not additional functional interview questions.
5. Review the draft: WHAT and WHY only, numbered EARS RF requirements, explicit
   exclusions, observable completion criteria linked to RF IDs, and compliance
   with all applicable constitutional principles. Reject implementation code,
   architecture designs, task lists, unconfirmed requirements, or claimed test
   results. Criteria must remain observable without automatically imposing tests;
   reflect the selected feature policy without designing implementation.
6. Present the complete draft and any unresolved issues. Ask the user to approve
   BOTH its exact content and saving it at the agreed path. A correction or a
   request for a spec does not authorize saving. Do not approve behavior-defining
   gaps as ready. Keep drafts in the conversation until this gate passes.
7. Relay that explicit approval to `sdd-spec`, including the exact approved
   content and destination, and direct it to create or revise that file only.
   Do not use `android-worker` as the default writer for this workflow. Require another
   destination and existing-content check before editing; if circumstances have
   changed since approval, stop for review rather than overwrite local work.
8. Read the resulting spec and verify its destination, approved content, EARS
   numbering, completion criteria, and constitutional compliance. Report what
   was actually checked. Saving a spec does not implement the feature, validate
   its tests, or authorize creating a plan or tasks.
