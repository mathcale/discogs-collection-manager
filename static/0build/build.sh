#!/usr/bin/env bash
set -euo pipefail

TAG="${1:?usage: build.sh <tag> <output dir>}"
OUT="${2:?usage: build.sh <tag> <output dir>}"
HERE="$(cd "$(dirname "$0")" && pwd)"
OUT="$(cd "$OUT" && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

case "$(uname -s)" in
  Darwin) LIGHTNINGCSS_OS=darwin ;;
  *) LIGHTNINGCSS_OS=linux ;;
esac

case "$(uname -m)" in
  arm64|aarch64) LIGHTNINGCSS_ARCH=arm64 ;;
  *) LIGHTNINGCSS_ARCH=x64 ;;
esac

if [ "$LIGHTNINGCSS_OS" = linux ]; then
  if ldd --version 2>&1 | grep -qi musl; then
    LIGHTNINGCSS_ABI=-musl
  else
    LIGHTNINGCSS_ABI=-gnu
  fi
else
  LIGHTNINGCSS_ABI=
fi

LIGHTNINGCSS="node_modules/lightningcss-cli-${LIGHTNINGCSS_OS}-${LIGHTNINGCSS_ARCH}${LIGHTNINGCSS_ABI}/lightningcss"

git clone --depth 1 --branch "$TAG" https://github.com/0builddotdev/0build "$WORK/0build"
cd "$WORK/0build"
npm ci --ignore-scripts --no-audit --no-fund

cp "$HERE/trim-css.mjs" "$HERE/pin.config.js" .
mkdir -p src/js
cp "$HERE/pin-entry.ts" src/js/pin-entry.ts

node_modules/.bin/vite -c pin.config.js build

REPO="$(cd "$HERE/../.." && pwd)"
{
  find "$REPO/internal" -name '*.templ' -print0 | xargs -0 cat
  cat "$REPO/static/app.css" dist/js/uikit.min.js out/pin.iife.js
} | grep -oE 'z-[a-zA-Z0-9_-]+' | sort -u > used-classes.txt

node trim-css.mjs . kit.trim.src.css used-classes.txt
"$LIGHTNINGCSS" --minify kit.trim.src.css -o kit.trim.min.css

cp kit.trim.min.css "$OUT/kit.min.css"
cp dist/js/uikit.min.js "$OUT/uikit.min.js"
cp out/pin.iife.js "$OUT/pin.min.js"
