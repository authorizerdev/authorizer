#!/bin/sh
# Wait until a Cassandra/ScyllaDB container actually serves CQL.
#
# wait-for-test-dbs.sh probes with `nc -z`, which on Docker Desktop connects to
# the published-port proxy rather than to the server: it succeeds the instant
# `docker run -p` returns, while Cassandra is still ~50s from binding its
# native transport. Measured — `nc -z 127.0.0.1 9143` succeeded at t=1s while
# `cqlsh` inside the same container still failed at t=3s. So for these two
# backends the port probe proves nothing and the storage tests can start against
# a database that is still booting. This asks the server itself, from inside the
# container, which a proxy cannot answer for.
set -e

CONTAINER="$1"
# Its own knob, deliberately not TEST_DB_WAIT_TIMEOUT: that one means "how long
# to wait for a port to open" and is routinely set low. Overloading it would
# make TEST_DB_WAIT_TIMEOUT=30 fail every cold Cassandra start, which needs ~50s.
TIMEOUT_SECONDS="${TEST_CQL_WAIT_TIMEOUT:-180}"

if [ -z "$CONTAINER" ]; then
  echo "usage: wait-for-cql.sh <container-name>" >&2
  exit 2
fi

# A wall-clock deadline, not a count of sleeps: each probe itself takes seconds
# during bootstrap, so counting sleeps made the stated timeout an undercount and
# the real ceiling open-ended.
started=$(date +%s)
deadline=$(( started + TIMEOUT_SECONDS ))

# cqlsh's own timeouts bound each attempt. `timeout(1)` would be the obvious
# tool but it is not present on macOS, where this runs most often.
CQLSH_TIMEOUTS="--connect-timeout=5 --request-timeout=5"

echo "waiting for CQL in $CONTAINER (up to ${TIMEOUT_SECONDS}s)..."
while :; do
  # Fail fast rather than burn the whole timeout in silence when the container
  # is gone or died mid-wait — an OOM kill at MAX_HEAP_SIZE=512M otherwise looks
  # exactly like a slow boot, for three minutes.
  # tr strips the newline docker emits even on failure, which otherwise made
  # the value "\nmissing" and skipped the `missing` branch below.
  state=$(docker inspect -f '{{.State.Status}}' "$CONTAINER" 2>/dev/null | tr -d '[:space:]')
  [ -n "$state" ] || state=missing
  case "$state" in
    running) ;;
    missing)
      echo "  no such container: $CONTAINER" >&2
      exit 1
      ;;
    *)
      echo "  $CONTAINER is '$state', not running — it died during startup" >&2
      docker logs --tail 20 "$CONTAINER" 2>&1 | sed 's/^/    /' >&2
      exit 1
      ;;
  esac

  # shellcheck disable=SC2086  # word splitting of the flag list is intended
  if docker exec "$CONTAINER" cqlsh $CQLSH_TIMEOUTS \
       -e "SELECT release_version FROM system.local" >/dev/null 2>&1; then
    echo "  $CONTAINER serving CQL (after $(( $(date +%s) - started ))s)"
    exit 0
  fi

  if [ "$(date +%s)" -ge "$deadline" ]; then
    echo "  $CONTAINER never served CQL within ${TIMEOUT_SECONDS}s" >&2
    docker logs --tail 20 "$CONTAINER" 2>&1 | sed 's/^/    /' >&2
    exit 1
  fi
  sleep 2
done
