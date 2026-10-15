#!/bin/sh
# Wait until a Cassandra/ScyllaDB container actually serves CQL.
#
# wait-for-test-dbs.sh probes with `nc -z`, which on Docker Desktop connects to
# the published-port proxy rather than to the server: it succeeds the instant
# `docker run -p` returns, while Cassandra is still minutes from binding its
# native transport. Measured — `nc -z 127.0.0.1 9143` succeeded at t=1s while
# `cqlsh` inside the same container still failed at t=3s.
#
# So for these two backends the port probe proves nothing, and the storage tests
# can start against a database that is still booting. `go test ./...` usually
# spends long enough on other packages to hide it; a narrower target does not.
# This asks the server itself, from inside the container, which cannot be
# answered by a proxy.
set -e

CONTAINER="$1"
TIMEOUT_SECONDS="${TEST_DB_WAIT_TIMEOUT:-180}"

if [ -z "$CONTAINER" ]; then
  echo "usage: wait-for-cql.sh <container-name>" >&2
  exit 2
fi

echo "waiting for CQL in $CONTAINER..."
elapsed=0
while [ "$elapsed" -lt "$TIMEOUT_SECONDS" ]; do
  if docker exec "$CONTAINER" cqlsh -e "SELECT release_version FROM system.local" >/dev/null 2>&1; then
    echo "  $CONTAINER serving CQL (after ${elapsed}s)"
    exit 0
  fi
  sleep 2
  elapsed=$((elapsed + 2))
done
echo "  $CONTAINER never served CQL within ${TIMEOUT_SECONDS}s" >&2
exit 1
