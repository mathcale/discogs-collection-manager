# Copilot Workspace Instructions

## Issue Tracker

This project is tracked in Linear. See [.agents/linear.md](.agents/linear.md) for the full project reference, PRD link, issue list, and dependency graph.

- **Linear project:** https://linear.app/automatta-tech/project/discogs-collection-manager-1c0eb5085335
- **Team prefix:** `ATMT`
- **PRD:** https://linear.app/automatta-tech/document/prd-discogs-collection-manager-4d2e202ef767

When filing issues, referencing work, or picking up tasks, always use the Linear project above.

## Tooling

- **Git / GitHub:** Use the terminal only. Do not use the GitKraken MCP tools for any git or GitHub operations.

## Branching

- All implementation work must happen on a feature branch — never commit directly to `main`.
- Branch naming: `feat/<short-description>` for features, `fix/<short-description>` for bug fixes, `chore/<short-description>` for maintenance tasks.
- On finishing implementing a task, open a PR to `main` and assign it to the task in Linear. The PR title should match the Linear issue title.

## Tech Stack

- **Language:** Go
- **Router:** `chi`
- **Templates:** `a-h/templ`
- **Interactivity:** HTMX (no frontend framework)
- **Database:** SQLite via `database/sql` + `mattn/go-sqlite3`
- **Migrations:** `golang-migrate`
- **Email:** `Mailer` interface, Resend implementation
- **Discogs auth:** OAuth 1.0a via `dghubble/oauth1`
- **Live reload:** `cosmtrek/air`
- **Tests:** `testify/suite` against `net/http/httptest`

## Code Style

- **No comments** — names must be self-explanatory. Only add a comment when logic is genuinely non-obvious and cannot be clarified by renaming. Never add docstrings, type-description comments, or function-summary comments.
- **No over-engineering** — only implement what is directly asked for. No speculative abstractions, helpers for one-off operations, or extra error handling for impossible cases.

## Package Structure

```
internal/
  web/       # HTTP layer: router (chi routes + middleware), Server (listen + graceful shutdown)
  pkg/       # Infrastructure utilities with no business logic: kernel/, migrate/, ...
  handler/   # HTTP handlers — one file per feature area
  templates/ # a-h/templ components
```

`cmd/server/main.go` only orchestrates: `kernel.New()` → `web.NewServer(k).Run()`. Nothing else.

## Architecture Conventions

- All database access is behind repository interfaces (`UserRepository`, `SessionRepository`, `OAuthTokenRepository`). Inject SQLite implementations at startup.
- The Discogs HTTP client is hidden behind a `DiscogsClient` interface. Inject it — never instantiate it inside handlers.
- All email sending goes through a `Mailer` interface. Never call Resend directly from business logic.
- Tests use the HTTP handler boundary as the primary seam: spin up the full handler tree with an in-memory SQLite DB and mock `DiscogsClient` + `Mailer`.
- No specific file paths or code snippets in issue descriptions — they go stale.
