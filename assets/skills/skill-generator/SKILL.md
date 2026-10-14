---
name: skill-generator
description: Create or refine reusable {{product}} skills, either personal (global) or shared with a repository. Use when asked to create, adapt, or improve a skill.
---

# Skill generator

Create focused, reusable skills. Skills live in one of two places:

| Scope | Location | Who gets it |
|---|---|---|
| Personal | `{{skills_user_dir}}/<name>/SKILL.md` | Only this user, in every project |
| Repository | `<repo>/{{skills_repo_dir}}/<name>/SKILL.md` | Everyone who works in the repository (committed with it) |

Ask which scope the user wants when the request does not make it clear. A repository skill
overrides a personal skill with the same name.

## When to use

- A recurring workflow needs explicit instructions for an AI assistant.
- A developer asks to create, adapt, or improve a skill.
- Project-specific rules need to be captured as a reusable capability rather than repeated in prompts.

If the need is one-off or already covered by an existing skill, suggest using or refining that skill instead.

## Before writing

1. List the existing skills in both locations and check for overlapping capabilities.
2. Read up to three relevant existing `SKILL.md` files to follow local conventions.
3. Inspect the real project files relevant to the requested capability. Do not invent modules,
   paths, toolchains, or team conventions.
4. Identify the intended outcome, trigger, inputs, outputs, boundaries, and tools. Ask only for
   requirements that cannot reasonably be inferred.
5. Propose a narrowly scoped skill and explain what it will do before creating files.

## Naming and description

- Use a descriptive `kebab-case` directory name, 1–64 characters, matching
  `^[a-z0-9]+(-[a-z0-9]+)*$`. The directory name is the skill ID.
- Ensure the name does not conflict with an existing skill.
- Start the file with YAML frontmatter containing `name` and `description`.
- Make `description` a concise, third-person sentence covering both what the skill does and when
  to load it, with the terms a developer is likely to use when asking for it. Skills without a
  description are not offered to the model.

## Skill content

Include only sections that make the workflow actionable:

1. **Purpose and trigger:** when to use the skill, and when not to use it.
2. **Prerequisites:** real local files, conventions, or external services that must be checked.
3. **Workflow:** ordered decisions and actions, including stop conditions and error handling.
4. **Output:** what the user should receive and how to verify it.
5. **What NOT to do:** concrete actions the skill must not take, assumptions it must not make, and
   operations that require explicit user authorization.

Every new skill must include a `## What NOT to do` section specific to its risks. Prefer short
instructions and links to existing documentation over copying documents. Keep supporting files
(`references/`, `scripts/`) beside `SKILL.md` only when the workflow needs them; paths inside the
skill are relative to its directory.

## Create or update a skill

1. Draft the name, description, and outline from the request and the repository evidence.
2. If a significant design choice is open, clarify it first; otherwise build the smallest useful version.
3. Write `<location>/<name>/SKILL.md`, preserving useful existing content when updating.
4. Verify frontmatter, name, trigger, location, and the `What NOT to do` section.
5. Tell the user that running {{product}} sessions pick up new skills in a new session, and that a
   repository skill reaches the team only after it is committed and merged.

## What NOT to do

- Do not create duplicate skills for the same responsibility.
- Do not copy instructions from another repository without checking and adapting them.
- Do not add unrelated agents, commands, or infrastructure to make a skill look operational.
- Do not require scripts or shared files that do not exist.
- Do not commit or push a repository skill unless the user asks.
- Do not omit `## What NOT to do` or fill it with generic boilerplate.
