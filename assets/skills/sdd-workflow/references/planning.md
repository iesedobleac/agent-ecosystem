# Technical Planning

Use this workflow when the user requests a technical plan for a specific spec.
Do not automatically create a plan after saving a spec, implement the feature,
or create tasks. `sdd-clarify` remains optional and user-requested.

1. Establish the root, exact spec version, and its feature folder. Respect
   AGENTS.md instructions already available without needless rereading; discover
   applicable instructions if absent. Explicitly read the constitution and spec,
   and inspect relevant architecture and documentation. Report missing sources
   or conflicts rather than filling gaps with assumptions.
2. The sole destination is `<project-root>/ai/specs/<spec-name>/plan.md`, beside the
   source spec. Validate the root and safe folder name, inspect any existing plan,
   and preserve local work. Do not silently choose between spec versions.
3. Delegate to `sdd-plan` with sources, root, spec and destination, relevant
   instructions, confirmed decisions, scope, constraints, and existing plan if
   any. Initially authorize analysis and a draft only: the child writes the full
   draft to `@@OPENCODE_TMP@@/sdd-drafts/<feature>-plan-<n>.md` and returns a
   summary (about 2k tokens or less) plus the path.
4. Relay one necessary technical question at a time, with a recommended option
   and relevant alternative when a decision is needed. No fixed question limit
   applies here. Explore first and do not repeat resolved questions. Maintain a
   cumulative decision-and-answer record across child sessions. Functional gaps
   return to the appropriate spec procedure (`specification.md` or
   `requirement-change.md`) without resetting or bypassing its question limit;
   do not preload those procedures unless the work actually needs to return there.
5. Review modules and responsibilities, data model where relevant, justified DT
   decisions and discarded alternatives, policy-appropriate verification, and a
   table covering every RF. Check fidelity to the spec and constitution,
   shared/platform scope, verified versus proposed architecture, and unresolved
   coverage. Reject code, executable artifacts, task lists, invented business
   rules, or claimed test execution. Test strategy and complete-suite gates apply
   only in `tests`/`tdd`; `none` records non-test evidence without unagreed
   substitute checks.
6. Obtain explicit decisions for proposed dependencies, services, architecture,
   contract, or data changes as required by project rules. Present the complete
   draft path with its summary and request approval of BOTH its exact content
   and saving it. Do not
   present behavior-defining gaps or missing RF coverage as ready.
7. Relay that approval and the draft path to `sdd-plan`, authorizing only the agreed
   plan file (it copies the approved draft content). Require another source/destination check. Existing-plan replacement
   requires explicit authorization; changed sources or local content must stop
   writing for review. Do not use `android-worker` as the default writer here.
8. Read the saved plan and check destination, approved content, RF coverage, and
   constitutional compliance. Report actual checks and limitations. Plan approval
   does not authorize implementation, task creation, commits, or publication.
