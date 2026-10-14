# Project Constitution

Use this workflow only when the user requests creating or reviewing a constitution;
do not start it automatically for every project or change.

1. Establish the project root and use read-only discovery tools to search for
   existing `constitution.md` files recursively, including hidden project
   directories and filename capitalization variants. Do not rely on a root-only
   or incomplete search. The canonical path is
   `<project-root>/docs/constitution.md`.
2. If any constitution exists outside the canonical path, report its exact path
   and stop immediately, even if a canonical file also exists. Do not invoke
   `sdd-constitution`, continue the interview, draft, write, move, or delegate any
   further constitution work until the user decides how to handle the location.
   If the root or search completeness is uncertain, report the blocker and stop.
3. If none exists, the proposed target is the canonical path. If only the
   canonical file exists, use it. Delegate to `sdd-constitution` with the root,
   location findings, objective, applicable instructions, existing document if
   any, and confirmed discovery context. The child must repeat the location check
   as a safeguard; honor any blocker it reports.
4. Relay its questions to the user in batches of up to three before drafting or
   proposing changes when important information is missing or contradictory.
   Continue the child session with answers, or provide a complete context summary
   in a new session. Do not repeat answered questions or invent principles.
5. Present the draft and unresolved issues. Obtain explicit approval of both the
   content and saving it. Then delegate writing to `android-worker`, providing the
   approved content and target path and requiring another location check before
   editing. If a noncanonical constitution appears, the writer must report its
   path and stop without changing anything. By default, save only to
   `<project-root>/docs/constitution.md`; any location exception requires an
   explicit user decision. Preserve existing work and review the resulting file.

`sdd-constitution` proposes principles; it does not execute commands or write
files. Read-only inspection is allowed before questions, but no unresolved
consequential decision may be filled in without user confirmation.
