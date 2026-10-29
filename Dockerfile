FROM golang:1.27-alpine AS builder

RUN apk add --no-cache gcc musl-dev curl

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/a-h/templ/cmd/templ@$(go list -m -f '{{.Version}}' github.com/a-h/templ) && templ generate
RUN CGO_ENABLED=1 GOOS=linux go build -o bin/discogs-collection-manager ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates sqlite su-exec \
    && addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=builder /app/bin/discogs-collection-manager ./
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/static ./static

COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh \
    && mkdir -p /app/data && chown -R app:app /app/data

EXPOSE 8080
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["./discogs-collection-manager"]
