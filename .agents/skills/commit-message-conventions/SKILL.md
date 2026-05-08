---
name: commit-message-conventions
description: "Use when the user asks Codex to create a git commit, commit at, commitle, prepare a commit message, name a commit, or otherwise commit project changes. Enforces the repository convention feat(scope): title and related commit workflow rules."
---

# Commit Message Conventions

## Required Format

Use this commit message format:

```text
feat(scope): title
```

- Use `feat` as the type unless the user explicitly provides another type.
- Use a short lowercase `scope` that names the changed area, such as `repo`, `api`, `web`, `docs`, `listener`, `db`, or `auth`.
- Write `title` in concise English, imperative or noun-style, with no trailing period.
- Keep the first line under 72 characters when practical.

## Commit Workflow

- Before committing, inspect `git status --short` and stage only intended files.
- Do not push unless the user explicitly asks.
- Do not include secrets, `.env`, virtual environments, logs, caches, or generated dependency folders.
- If unrelated user changes exist, leave them unstaged unless the user explicitly includes them.
- Prefer one focused commit per completed unit of work.

## Examples

```text
feat(repo): scaffold monorepo docs
feat(api): add kick chat ingestion
feat(web): add message search interface
feat(docs): document local startup
```
