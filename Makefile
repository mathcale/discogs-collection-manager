.PHONY: build deps dev migrate migrate-down test coverage generate fonts 0build

BINARY_NAME=discogs-collection-manager
MAIN_PKG=./cmd/server
MIGRATIONS_DIR=./migrations
DB_PATH=./data/discogs.db

ZEROBUILD_VERSION=v0.7.8
ZEROBUILD_OUTPUT=static/vendor/0build

GOBIN=$(shell go env GOPATH)/bin
MIGRATE=$(GOBIN)/migrate

0build:
	mkdir -p $(ZEROBUILD_OUTPUT)
	tools/0build/build.sh $(ZEROBUILD_VERSION) $(ZEROBUILD_OUTPUT)

fonts:
	mkdir -p static/fonts
	curl -fsSL "https://fonts.bunny.net/inter/files/inter-latin-variable-full-normal.woff2" -o static/fonts/inter-variable.woff2
	curl -fsSL "https://fonts.bunny.net/jetbrains-mono/files/jetbrains-mono-latin-400-normal.woff2" -o static/fonts/jetbrains-mono.woff2

build: generate
	go build -o bin/$(BINARY_NAME) $(MAIN_PKG)

deps:
	go mod download
	go install github.com/a-h/templ/cmd/templ@$$(go list -m -f '{{.Version}}' github.com/a-h/templ)
	go install github.com/air-verse/air@latest
	go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
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
