---
name: claude-pr-review
description: Request the Claude CLI to review a Pull Request, apply any review comments, and repeat the loop until the review is green (no further comments).
argument-hint: <PR number or branch to review>
---

## Goal

Drive a Claude CLI code-review loop over a Pull Request on this repository until the reviewer reports no further comments (everything is green).

Applies to: README.md, Go source, GitHub Actions workflows, and any other file changed in the PR.

## Referenced rules (always enforced)

- `@rules/good-practices.md` — single responsibility, no irrelevant comments, env vars over hard-coded values.
- `@rules/golang.md` — idiomatic Go, `goimports` formatting, testify assertions, no conditionals in tests, descriptive `camelCase` test names.
- `@rules/git.md` — prefer GitHub MCP over git CLI for GitHub operations; local staging/committing via git CLI only.
- `@rules/commit-message.md` — conventional commits format (`feat:`, `fix:`, `refactor:`, `chore:`, `docs:`), short and descriptive, no Co-Authored-By.
- `@rules/pull-request.md` — PR titles in conventional commits format, summary/changes/testing in description, small focused PRs, self-review before requesting review.
- `@rules/system-compatibility.md` — detect environment before writing scripts; never assume OS/arch.

## Workflow

### 1. Ensure the PR exists

Confirm a Pull Request exists for the current feature branch. Create one via GitHub MCP (`mcp__github__create_pull_request`) if missing, following `@rules/pull-request.md`.

### 2. Request the Claude CLI review

Ask the Claude CLI to review the PR diff (`gh pr diff <number>` / `mcp__github__pull_request_read method:get_diff` / `mcp__github__pull_request_read method:get_files`). The reviewer must:

- List approved items.
- List required changes with `file:line` references.
- State whether the round is **"green"** (no further comments) or **"needs fixes"**.

### 3. Apply fixes

For every comment the Claude CLI raises, edit the relevant file(s). After each round:

- `go build ./...`
- `go test ./...`
- `golangci-lint run --timeout=5m ./...` (note: may fail solely due to golangci-lint/Go toolchain export-data mismatch; treat build+test green as the primary signal and report the mismatch separately).

### 4. Re-request review

Push the fixes and ask the Claude CLI to review the PR again.

### 5. Loop

Repeat steps 3–4 until the Claude CLI reports no further comments (green).

### 6. Report

Tell the user: PR URL, number of review rounds, verification results (build/test), and final green status.

## Notes

- Do not commit/push unless the user has authorized it.
- Keep each review round small and focused; fix one comment set at a time.
- If a comment is about a rule mismatch (e.g. lint tool version), note it explicitly rather than treating it as a code defect.
