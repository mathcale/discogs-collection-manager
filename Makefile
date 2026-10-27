.PHONY: build deps dev migrate migrate-down test coverage generate css fonts tailwind

BINARY_NAME=discogs-collection-manager
MAIN_PKG=./cmd/server
MIGRATIONS_DIR=./migrations
DB_PATH=./data/discogs.db

TAILWIND_VERSION=v4.3.2
TAILWIND_BINARY=bin/tailwindcss
CSS_INPUT=static/app.css

GOBIN=$(shell go env GOPATH)/bin
MIGRATE=$(GOBIN)/migrate
CSS_OUTPUT=static/app.min.css

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Darwin)
  TAILWIND_OS=macos
else
  TAILWIND_OS=linux
endif

ifeq ($(UNAME_M),arm64)
  TAILWIND_ARCH=arm64
else ifeq ($(UNAME_M),aarch64)
  TAILWIND_ARCH=arm64
else
  TAILWIND_ARCH=x64
endif

tailwind:
	mkdir -p bin
	curl -fsSL "https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(TAILWIND_OS)-$(TAILWIND_ARCH)" -o $(TAILWIND_BINARY)
	chmod +x $(TAILWIND_BINARY)

fonts:
	mkdir -p static/fonts
	curl -fsSL "https://fonts.bunny.net/inter/files/inter-latin-variable-full-normal.woff2" -o static/fonts/inter-variable.woff2
	curl -fsSL "https://fonts.bunny.net/jetbrains-mono/files/jetbrains-mono-latin-400-normal.woff2" -o static/fonts/jetbrains-mono.woff2

css:
	$(TAILWIND_BINARY) -i $(CSS_INPUT) -o $(CSS_OUTPUT) --minify

build: generate css
	go build -o bin/$(BINARY_NAME) $(MAIN_PKG)

deps:
	go mod download
	go install github.com/a-h/templ/cmd/templ@$$(go list -m -f '{{.Version}}' github.com/a-h/templ)
	go install github.com/air-verse/air@latest
	go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	$(MAKE) tailwind
	$(MAKE) fonts

generate:
	templ generate

dev:
	air

migrate:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "sqlite3://$(DB_PATH)" up

migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "sqlite3://$(DB_PATH)" down

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
