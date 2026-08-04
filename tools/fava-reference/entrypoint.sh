#!/bin/sh
set -eu

: "${ORANGECOUNT_REFERENCE_IMAGE_ID:=unknown}"
: "${FAVA_REFERENCE_OUTPUT:=/output}"
mkdir -p "$FAVA_REFERENCE_OUTPUT"

fava --host 127.0.0.1 --port 5001 /fixtures/main.bean > /tmp/fava.log 2>&1 &
fava_pid=$!
cleanup() {
  kill "$fava_pid" 2>/dev/null || true
  wait "$fava_pid" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

ready=0
i=0
while [ "$i" -lt 60 ]; do
  if curl -fsS http://127.0.0.1:5001/ >/dev/null 2>&1; then ready=1; break; fi
  i=$((i + 1))
  sleep 1
done
if [ "$ready" -ne 1 ]; then
  echo "Fava reference failed health check" >&2
  sed 's# /[^ ]*# [path]#g' /tmp/fava.log >&2 || true
  exit 1
fi

node /app/write-environment-lock.mjs "$FAVA_REFERENCE_OUTPUT/environment-lock.json" \
  "$ORANGECOUNT_REFERENCE_IMAGE_ID" /fixtures

cd /app/web
FAVA_BASE_URL=http://127.0.0.1:5001 \
  CHROMIUM_EXECUTABLE_PATH=/usr/bin/chromium \
  VISUAL_SNAPSHOT_DIR="$FAVA_REFERENCE_OUTPUT/screenshots" \
  VISUAL_ENV_LOCK="$FAVA_REFERENCE_OUTPUT/environment-lock.json" \
  node node_modules/@playwright/test/cli.js test \
    --config /app/web/playwright.config.mjs visual/fava-reference.spec.mjs \
    --update-snapshots "$@"
