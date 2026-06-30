FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go install github.com/a-h/templ/cmd/templ@latest && templ generate
RUN CGO_ENABLED=1 GOOS=linux go build -o bin/discogs-collection-manager ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates sqlite

WORKDIR /app
COPY --from=builder /app/bin/discogs-collection-manager ./
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/static ./static

RUN mkdir -p /app/data

EXPOSE 8080
CMD ["./discogs-collection-manager"]
