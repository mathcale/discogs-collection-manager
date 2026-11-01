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
- **Styling:** 0build UI kit, vendored in `static/vendor/0build/` and trimmed to the `z-*` classes in use. After adding a `z-*` class to a template or `static/app.css`, or bumping `ZEROBUILD_VERSION`, run `make 0build` and commit `static/vendor/0build/`. `TestVendoredZerobuildKit_HasEveryUsedClass` goes red when the kit is stale. Classes that UIkit JS builds at runtime never appear in templates, so add their prefix to the runtime list in `static/0build/trim-css.mjs`. 0build web components (`<z-input-pin>`, `<z-select>`) are bundled into `pin.min.js` from `static/0build/pin-entry.ts`; the trimmer scans that bundle, so their runtime classes are kept automatically.
- **Database:** SQLite via `database/sql` + `mattn/go-sqlite3`
- **Migrations:** `golang-migrate`
- **Email:** `Mailer` interface, Resend implementation
- **i18n:** `pkg/i18n`, English (default) and `pt-BR`. Wrap every user-facing string in `i18n.T` with the English text as key and add its pt-BR entry to `internal/pkg/i18n/locales/pt-BR.json`; `TestI18nCatalog_*` goes red otherwise.
- **Discogs auth:** OAuth 1.0a via `dghubble/oauth1`
- **Live reload:** `cosmtrek/air`
- **Tests:** `testify/suite` against `net/http/httptest`

## Code Style

- **No comments** — names must be self-explanatory. Only add a comment when logic is genuinely non-obvious and cannot be clarified by renaming. Never add docstrings, type-description comments, or function-summary comments.
- **No over-engineering** — only implement what is directly asked for. No speculative abstractions, helpers for one-off operations, or extra error handling for impossible cases.

## Package Structure

```plaintext
cmd/server/main.go                      core.New() → web.NewServer(k).Run()

internal/
  core/
    kernel.go                           env config, DB, migrations, clients, feature modules
    web/
      routes.go                         global middleware + mounts each module
      middleware.go                     RequestLogger only
      server.go
    clients/
      discogs/
      mailer/

  features/
    landing/        feature.go, handlers/, views/
    auth/           feature.go, handlers/, usecases/, models/, views/
    account/        feature.go, handlers/, usecases/, models/, views/
    scanner/        feature.go, handlers/, usecases/, views/

  ui/                                   BaseLayout, AppLayout, AlertError, icons, Render
  pkg/                                  ctxkey/  i18n/  logger/  migrate/

migrations/
```

`cmd/server/main.go` only orchestrates: `core.New()` → `web.NewServer(k).Run()`. Nothing else.

## Import Rules

| Package | May import |
|---|---|
| `core/web` | `core`, `pkg/*` |
| `core` (kernel) | feature **root** packages, `core/clients/*`, `pkg/*`, and `features/account/models` (only to build the Discogs client) |
| `core/clients/discogs` | `features/account/models`, `pkg/ctxkey` |
| feature root (`feature.go`) | its own subpackages, other features' `models`, `core/clients/*`, `chi` |
| `<f>/handlers` | own `usecases`, own `views`, `ui`, `pkg/ctxkey`, `pkg/i18n`, `core/clients/discogs` (error types only) |
| `<f>/usecases` | own `models`, other features' `models`, `core/clients/*`, `pkg/i18n` |
| `<f>/models` | stdlib only (plus crypto libs already in use) |
| `<f>/views` | `ui`, own `models`, `core/clients/discogs` (release types), `pkg/i18n` |
| `ui` | `templ`, `pkg/i18n` |

Hard rules:

- A feature never imports another feature's `handlers`, `usecases`, `views` or root package.
- Subpackages never import their own feature root.

## Architecture Conventions

- Handlers handle HTTP only (parsing, cookies, redirects, rendering). Business flows live in use cases; domain rules (password hashing, token generation, expiry checks) live in `models`.
- All database access is behind repository interfaces (for example `UserRepository`, `SessionRepository`, `OAuthTokenRepository`). Each feature builds its repositories from the injected `*sql.DB` in its `feature.go`.
- The Discogs HTTP client is hidden behind a `DiscogsClient` interface. Inject it — never instantiate it inside handlers or use cases.
- All email sending goes through a `Mailer` interface. Never call Resend directly from business logic.
- Tests use the HTTP handler boundary as the primary seam: each feature tests its `Module.Routes` with an in-memory SQLite DB and mock `DiscogsClient` + `Mailer`. Full-tree tests live in `core/web`.

### Feature modules

- Each feature has a root `feature.go` declaring a `Deps` struct and a `Module`. `Deps` holds only shared infrastructure (DB, clients, config values), never another feature's module or repository. A feature that needs another feature's data builds that feature's repository itself from the DB.
- `New(Deps)` wires repositories, use cases and the handler. `Routes` registers absolute paths; `core/web` only decides grouping and middleware.
- Repositories take the DB through their constructor, which returns the interface; implementations are unexported and have no `SQLite` prefix.
- Each use case lives in its own file with its input, output, sentinel errors, unexported local interfaces for dependencies owned by other features, the use case struct with exported dependency fields, and an `Execute` method. Use cases know nothing about HTTP and never call each other; shared logic goes in an unexported helper in a non-use-case file.
- Unexpected errors are returned as-is and handlers turn unrecognised ones into `500 internal error`. Discogs client errors must stay reachable with `errors.As`; when a use case needs to name the failed step, wrap with both the sentinel and the original error.
- Handler structs hold concrete use-case pointers named after the use case. Handler methods keep their action names; when a method name collides with a use-case field, the method gets a `Submit` suffix.
- No specific file paths or code snippets in issue descriptions — they go stale.
