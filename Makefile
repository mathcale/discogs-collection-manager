.PHONY: build deps dev migrate migrate-down test coverage generate

BINARY_NAME=discogs-collection-manager
MAIN_PKG=./cmd/server
MIGRATIONS_DIR=./migrations
DB_PATH=./data/discogs.db

build: generate
	go build -o bin/$(BINARY_NAME) $(MAIN_PKG)

deps:
	go mod download
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/cosmtrek/air@latest
	go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

generate:
	templ generate

dev:
	air

migrate:
	migrate -path $(MIGRATIONS_DIR) -database "sqlite3://$(DB_PATH)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "sqlite3://$(DB_PATH)" down

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
