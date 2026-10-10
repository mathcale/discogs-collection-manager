# Agent Instructions

## Issue Tracker

This project is tracked in Linear (team prefix `ATMT`). The project URL, PRD link, issue list, and dependency graph live in `.agents/linear.local.md`. Read it if it exists; otherwise ask for the Linear project.

When filing issues, referencing work, or picking up tasks, always use that Linear project.

## Tooling

- **Git / GitHub:** Use the terminal only. Do not use MCP tools for any git or GitHub operations.

## Branching

- All implementation work must happen on a feature branch — never commit directly to `main`.
- Branch naming: `feat/<short-description>` for features, `fix/<short-description>` for bug fixes, `chore/<short-description>` for maintenance tasks.
- On finishing implementing a task, open a PR to `main` and assign it to the task in Linear. The PR title should match the Linear issue title.

## Commits

- When allowed to commit, use a one-line semantic commit message: `type(scope): description`.
- When working on a ticket, the scope is the ticket ID. Example: `feat(ATMT-73): add barcode scanner view`.
- Otherwise, the scope is the component or feature. Examples: `chore(discogs): bump oauth1 to latest version`, `fix(auth): expire sessions on password reset`.
- Never mention an agent or model as co-author or in any attribution line (no `Co-Authored-By`, no "Generated with" footers) in commits or PRs.
- Always commit with the machine-configured gitconfig identity. Never override the author or committer (no `--author`, no `git config user.*` changes, no `GIT_AUTHOR_*` / `GIT_COMMITTER_*` env vars).

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

```plaintext
internal/
  app/           # Config + bootstrap: builds the App dependency container
    clients/     # External service clients: discogs/, mailer/
    web/         # HTTP layer: routes (chi + middleware), Server (listen + graceful shutdown)
  models/        # Domain structs, repository interfaces + SQLite implementations, domain rules
  controllers/   # Single Controller struct; one file per feature area
  views/         # a-h/templ components
  pkg/           # Infrastructure utilities with no business logic: ctxkey/, logger/, migrate/
```

`cmd/server/main.go` only orchestrates: `app.New()` → `web.NewServer(a).Run()`. Nothing else.

Import direction: `app/web` → `app`, `controllers` → `models`, `views`, `app/clients`. `app` and `controllers` never import `app/web`; `controllers` never imports `app`.

## Architecture Conventions

- Controllers handle HTTP only (parsing, cookies, redirects, rendering). Domain rules (password hashing, token generation, expiry checks) live in `models`.
- All database access is behind repository interfaces (`UserRepository`, `SessionRepository`, `OAuthTokenRepository`). Inject SQLite implementations at startup.
- The Discogs HTTP client is hidden behind a `DiscogsClient` interface. Inject it — never instantiate it inside controllers.
- All email sending goes through a `Mailer` interface. Never call Resend directly from business logic.
- Tests use the HTTP handler boundary as the primary seam: spin up the full handler tree with an in-memory SQLite DB and mock `DiscogsClient` + `Mailer`.
- No specific file paths or code snippets in issue descriptions — they go stale.
