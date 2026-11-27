#!/bin/sh
# Boot the committed docker-compose.yaml and prove the quick start works:
# the server comes up, runs as the image's non-root user (uid 1000), writes
# its SQLite database to the named volume, and keeps the data across a
# restart and a container recreate. Guards #930 / #325 (compose quick start
# broken by v2 flag-only config and by the non-root user lacking a writable
# data directory).
#
# Usage (from anywhere; runs against the repo root):
#   scripts/ci-compose-smoke.sh
#
# Docker Compose's own env vars are honoured, so a local run can avoid
# clashing with other stacks without touching the committed compose file:
#   COMPOSE_PROJECT_NAME=mysmoke \
#   COMPOSE_FILE=docker-compose.yaml:/path/to/ports-override.yaml \
#   AUTHORIZER_URL=http://localhost:18080 scripts/ci-compose-smoke.sh
#
# AUTHORIZER_URL is where the host reaches the server (default
# http://localhost:8080). It is also sent as the Origin header: with no
# --allowed-origins the CSRF check requires Origin to match the Host the
# request was sent to.
# COMPOSE_SMOKE_KEEP=1 leaves the stack running afterwards for debugging.
set -eu

cd "$(dirname "$0")/.."

: "${COMPOSE_PROJECT_NAME:=authorizer-compose-smoke}"
export COMPOSE_PROJECT_NAME
URL="${AUTHORIZER_URL:-http://localhost:8080}"
ORIGIN="$URL"
# Must match --admin-secret in docker-compose.yaml.
ADMIN_SECRET="change-me-admin-secret"
EMAIL="compose-smoke-$(date +%s)@example.com"
PASSWORD="Compose-Smoke-1!"
DB_PATH="/authorizer/data/authorizer.db"
EXPECTED_UID=1000

log() { printf '==> %s\n' "$*"; }
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

cleanup() {
  rc=$?
  if [ "$rc" -ne 0 ]; then
    printf '\n---- docker compose ps ----\n' >&2
    docker compose ps -a >&2 || true
    printf '\n---- docker compose logs ----\n' >&2
    docker compose logs --no-color >&2 || true
  fi
  if [ "${COMPOSE_SMOKE_KEEP:-0}" != "1" ]; then
    log "tearing down (docker compose down -v)"
    docker compose down -v --remove-orphans >/dev/null 2>&1 || true
  fi
  exit "$rc"
}
# Cleanup runs on EXIT only; signals just exit non-zero, which fires it.
# (Inside an INT/TERM handler $? is not the signal status, so routing
# signals straight to cleanup could make Ctrl-C exit 0.)
trap cleanup EXIT
trap 'exit 130' INT TERM

# Loop on the HTTP status rather than curl --retry: right after start the
# Docker port proxy accepts the connection and then resets it, which
# --retry-connrefused does not treat as retryable.
wait_healthy() {
  i=0
  while [ "$i" -lt 90 ]; do
    code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 "$URL/healthz" || true)
    if [ "$code" = "200" ]; then
      log "healthy after ~$((i * 2))s"
      return 0
    fi
    # Fail fast if the server already exited (e.g. it could not open the
    # database) instead of waiting out the full timeout.
    if [ -n "$(docker compose ps -q --status exited authorizer)" ]; then
      fail "authorizer container exited before becoming healthy"
    fi
    i=$((i + 1))
    sleep 2
  done
  fail "/healthz did not return 200 within 180s (last status: ${code:-none})"
}

graphql() {
  # $1 = JSON body, remaining args = extra curl args (headers)
  body=$1
  shift
  curl -sS --max-time 10 -X POST "$URL/graphql" \
    -H 'Content-Type: application/json' \
    -H "Origin: $ORIGIN" \
    "$@" \
    --data "$body"
}

assert_user_persisted() {
  # $1 = phase label
  # Filter by email so a reused volume with many users cannot push this one
  # off the first page.
  resp=$(graphql "{\"query\":\"query { _users(params: {query: \\\"$EMAIL\\\", pagination: {limit: 10, page: 1}}) { users { email } } }\"}" \
    -H "x-authorizer-admin-secret: $ADMIN_SECRET")
  case "$resp" in
    *'"errors"'*) fail "_users after $1 returned errors: $resp" ;;
    *"\"$EMAIL\""*) log "user $EMAIL still present after $1" ;;
    *) fail "user $EMAIL missing after $1: $resp" ;;
  esac
}

log "building and starting the compose stack (project $COMPOSE_PROJECT_NAME)"
start=$(date +%s)
docker compose up -d --build
log "build + start took $(($(date +%s) - start))s"
wait_healthy

log "signing up $EMAIL"
resp=$(graphql "{\"query\":\"mutation { signup(params: {email: \\\"$EMAIL\\\", password: \\\"$PASSWORD\\\", confirm_password: \\\"$PASSWORD\\\"}) { message } }\"}")
# MFA is on by default, so the response carries no user yet ("Proceed to mfa
# setup"); the account is still created, which _users below confirms.
case "$resp" in
  *'"errors"'*) fail "signup returned errors: $resp" ;;
  *'"signup":{"message"'*) log "signup ok: $resp" ;;
  *) fail "unexpected signup response: $resp" ;;
esac
assert_user_persisted "signup"

log "checking process uid and database ownership"
uid=$(docker compose exec -T authorizer id -u | tr -d '\r')
[ "$uid" = "$EXPECTED_UID" ] || fail "server runs as uid $uid, expected $EXPECTED_UID ($(docker compose exec -T authorizer id))"
docker compose exec -T authorizer test -f "$DB_PATH" ||
  fail "$DB_PATH does not exist in the container"
owner=$(docker compose exec -T authorizer stat -c '%u' "$DB_PATH" | tr -d '\r')
[ "$owner" = "$EXPECTED_UID" ] || fail "$DB_PATH owned by uid $owner, expected $EXPECTED_UID"
log "process uid $uid, $DB_PATH owned by uid $owner"

log "restarting the container"
docker compose restart
wait_healthy
assert_user_persisted "restart"

# A restart keeps the container's writable layer, so it would also pass if
# the database were written outside the volume. Recreating the container
# (down without -v keeps named volumes) proves the volume holds the data.
log "recreating the container (volume kept)"
docker compose down
docker compose up -d
wait_healthy
assert_user_persisted "container recreate"

log "PASS"
