#!/bin/sh
set -e

# Railway mounts volumes as root; fix ownership, then drop privileges.
DB_DIR=$(dirname "${DB_PATH:-/app/data/discogs.db}")
mkdir -p "$DB_DIR"
chown -R app:app "$DB_DIR"

exec su-exec app "$@"
