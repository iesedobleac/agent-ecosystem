# On-Demand Specification Clarification

Run this workflow only upon an explicit user request to review a spec for
ambiguities, contradictions, missing edge cases, or constitutional conflicts.
Do not invoke it automatically after `sdd-spec`, as a saving gate, or merely
because a specification exists. It can review an existing file or a draft.

1. Identify the project root, exact spec version, requested review scope,
   applicable AGENTS.md, constitution, relevant documentation, and confirmed
   decisions. If multiple versions exist, do not silently select one.
2. Delegate to `sdd-clarify` with that context and the expected findings-only
   report. Explicitly prohibit code, edits, solutions, interviews, and invented
   requirements. State that the review is user-requested, not automatic.
3. Check that findings have CL IDs, categories, locations, impact, evidence, and
   certainty. Reject suggested solutions, scope expansion, fabricated evidence,
   or claims of executed tests. Present the report and limitations to the user.
4. The user decides what to clarify or change. Do not automatically resolve
   findings, alter the spec, start an interview, or impose another review.
   A review request does not authorize any of those actions.
5. If the user separately requests revisions, pass their confirmed decisions to
   `sdd-spec` under `requirement-change.md` for an existing spec or
   `specification.md` for a new draft, loading only the relevant procedure then.
   Preserve the existing question ledger and cumulative six-question limit;
   findings are not answers and the reviewer cannot reset that limit. Any new
   questions belong to the remaining allowance. Report unresolved blockers when
   the limit is exhausted. Obtain approval of the revised content and saving it
   before writing.

The review report remains in the conversation. Absence of findings is not
approval, proof of perfection, or evidence of feature or platform validation.
Preserve discovery, constitution, and specification workflows unchanged except
for this explicit, optional review capability.
