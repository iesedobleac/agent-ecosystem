# Independent Specification Verification

Use only when the user requests or authorizes verification of a spec or an
explicit RF subset. It is not an automatic post-task step or a prerequisite for
saving documents. Do not treat an apply report as independent verification.

1. Establish the root, exact spec/version, corresponding plan/tasks, implementation
   context, testing mode, agreed verification, and full-spec or named-RF scope.
   Reuse the existing mode; do not repeat the policy question. Missing context or
   conflicting versions must be reported rather than silently chosen.
2. Delegate to `sdd-verify` with relevant instructions/constitution, confirmed
   decisions, reports/results and their source linkage, scope, allowed checks,
   and expected RF matrix. Prohibit edits, fixes, task marking, delegation, and
   unagreed checks. Shell permission does not authorize tests in `none`.
3. Require per-RF implementation, actual evidence/coverage, and classification:
   fulfilled, unfulfilled, or not verifiable. Reject conclusions based solely on
   green tests, checkboxes, or implementer claims. Require freshness to the current
   source and explicit distinction between executed, reported, static, manual,
   skipped, cached, or absent-test evidence.
4. Review separate functional and verification-policy verdicts. Any demonstrated
   RF violation means functional unfulfilled; unsupported RFs without a proven
   violation mean inconclusive. A red required suite blocks tested acceptance,
   not necessarily a specific RF. Do not interpret unknowns as approval. In `none`,
   make clear that any fulfillment conclusion is not executed-test validation.
5. Present scope, matrix, evidence, limitations, and justified verdict. A partial
   review says nothing definitive about omitted RFs or the full spec. Confirm
   overall acceptance only with sufficient functional and agreed-check evidence.
6. Stop and await the user's decision. A verification request never authorizes
   repairs, new tasks, changes to criteria, checkboxes, source documents, or commits.
   Any requested correction must be separately authorized and delegated under the
   relevant workflow. Keep the report in the conversation, not a new file.
