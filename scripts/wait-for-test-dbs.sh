#!/bin/sh
# Wait until every test database is actually accepting connections.
#
# test-docker-up used to end in a bare `sleep 5`, which is not a readiness
# check — it is a guess. ScyllaDB in particular routinely needs 30-60s before
# it accepts CQL, so on a cold or loaded machine the storage tests started
# against a database that was still booting and failed for reasons that had
# nothing to do with the code. That produces the worst kind of red build: real
# looking, unreproducible, and eventually ignored.
#
# Bounded so a genuinely broken container fails the run instead of hanging CI.
set -e

TIMEOUT_SECONDS="${TEST_DB_WAIT_TIMEOUT:-120}"

# name:port pairs. Couchbase is absent on purpose — scripts/couchbase-test.sh
# already provisions and waits for it.
SERVICES="redis:6380 postgres:5434 mongodb:27017 scylladb:9042 arangodb:8529 dynamodb:8000"

wait_for_port() {
  name="$1"
  port="$2"
  elapsed=0
  while [ "$elapsed" -lt "$TIMEOUT_SECONDS" ]; do
    # nc is present on macOS and every CI image this repo builds on. -z is a
    # connect-only probe: no bytes are written, so it cannot confuse a server
    # that is mid-handshake.
    if nc -z 127.0.0.1 "$port" 2>/dev/null; then
      echo "  $name ready on :$port (after ${elapsed}s)"
      return 0
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done
  echo "  $name NOT ready on :$port after ${TIMEOUT_SECONDS}s" >&2
  return 1
}

echo "waiting for test databases..."
rc=0
for svc in $SERVICES; do
  name="${svc%%:*}"
  port="${svc##*:}"
  wait_for_port "$name" "$port" || rc=1
done

if [ "$rc" -ne 0 ]; then
  echo "one or more test databases never became ready; aborting rather than running against a half-booted stack" >&2
  exit 1
fi
echo "all test databases ready"
