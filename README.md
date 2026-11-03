# Discogs Collection Manager

A web app for managing your Discogs record collection, with barcode scanning to add releases quickly.

## Tech Stack

- **Language:** Go
- **Router:** `chi`
- **Templates:** `a-h/templ`
- **Interactivity:** HTMX
- **Database:** SQLite
- **Styling:** 0build (vendored, trimmed)

## Prerequisites

- Go 1.24+
- `make`
- Node.js (only needed to rebuild the vendored 0build kit with `make 0build`)
- `curl`
- [`mkcert`](https://github.com/FiloSottile/mkcert) (only needed for HTTPS / camera scanning - see below)

## Local Development

### 1. Install dependencies

This downloads Go modules and installs the required tools (`templ`, `air`, `migrate`), and fonts:

```sh
make deps
```

### 2. Configure environment

Copy the example env file and fill in the values:

```sh
cp .env.example .env
```

| Variable | Description |
| --- | --- |
| `APP_BASE_URL` | Public base URL of the app (used in emails and OAuth). |
| `ENVIRONMENT` | `development` enables extra logging (e.g. LAN address on startup). |
| `DB_PATH` | Path to the SQLite database file. |
| `MIGRATIONS_PATH` | Path to the migrations directory. |
| `TLS_CERT_FILE` / `TLS_KEY_FILE` | Paths to TLS cert/key. Leave empty for plain HTTP. See [HTTPS](#https-for-camera-scanning). |
| `DISCOGS_CONSUMER_KEY` / `DISCOGS_CONSUMER_SECRET` | Credentials from your [Discogs developer app](https://www.discogs.com/settings/developers). |
| `DISCOGS_CALLBACK_URL` | OAuth callback, e.g. `http://localhost:8080/discogs/callback`. |
| `RESEND_API_KEY` / `RESEND_FROM` | [Resend](https://resend.com) credentials for sending email. |
| `TWO_STEP_AUTH_ENABLED` | Require an emailed one-time code after the password at login. Defaults to `true`. |
| `SIGNUPS_ENABLED` | Allow new accounts to be registered. Defaults to `true`. |

The `.env` file is loaded automatically by `air`.

### 3. Run migrations

```sh
make migrate
```

### 4. Start the dev server

```sh
make dev
```

This runs `air`, which rebuilds and restarts the server on file changes. By default the app is available at http://localhost:8080.

## HTTPS (for camera scanning)

Barcode scanning uses the device camera via the browser's `getUserMedia` API, which only works in a **secure context** (HTTPS), except on `localhost`. To test scanning from a phone or another device on your LAN, you need to serve the app over HTTPS with a locally-trusted certificate.

### 1. Install mkcert and its local CA

```sh
brew install mkcert       # macOS; see mkcert docs for other platforms
mkcert -install
```

### 2. Generate a certificate

Include `localhost` plus your machine's LAN IP (shown in the server startup log when `ENVIRONMENT=development`):

```sh
mkdir -p certs
mkcert -cert-file certs/localhost.pem -key-file certs/localhost-key.pem localhost 127.0.0.1 <your-lan-ip>
```

> To let other devices (e.g. your phone) trust the certificate, install the mkcert root CA on them too. Run `mkcert -CAROOT` to find the CA file to transfer.

### 3. Point the app at the certificate

In your `.env`:

```sh
TLS_CERT_FILE="./certs/localhost.pem"
TLS_KEY_FILE="./certs/localhost-key.pem"
APP_BASE_URL="https://localhost:8080"
DISCOGS_CALLBACK_URL="https://localhost:8080/discogs/callback"
```

Restart `make dev`. The server now listens on HTTPS and logs its `https://<lan-ip>:8080` address, which you can open from other devices on the same network.

## Useful Commands

| Command | Description |
| --- | --- |
| `make deps` | Install all dependencies and tools. |
| `make dev` | Run the dev server with live reload. |
| `make build` | Generate templates and build the binary. |
| `make generate` | Regenerate `templ` templates. |
| `make 0build` | Rebuild the vendored 0build kit (needs Node). Run it after using a new `z-*` class or bumping `ZEROBUILD_VERSION`, then commit `static/vendor/0build/`. `go test ./...` fails while the kit is stale. |
| `make migrate` | Apply database migrations. |
| `make migrate-down` | Roll back the last migration. |
| `make test` | Run the test suite. |
| `make coverage` | Run tests and generate an HTML coverage report. |

## Docker

To run the app in a container:

```sh
docker compose up --build
```

The app is exposed on http://localhost:8080 with the SQLite database persisted in a named volume.
