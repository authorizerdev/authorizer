#!/bin/sh
# Wait until a SQL test container actually serves queries.
#
# Same reason wait-for-cql.sh exists: wait-for-test-dbs.sh probes with `nc -z`,
# which on Docker Desktop connects to the published-port proxy and succeeds the
# instant `docker run -p` returns. For these two engines that proves nothing —
# MySQL binds 3306, runs its init scripts and then *restarts*, and CockroachDB
# accepts TCP before it is accepting SQL clients. Both windows are long enough
# for the storage tests to start against a database that is not there yet. This
# asks the server itself, from inside the container, which a proxy cannot answer.
set -e

CONTAINER="$1"
ENGINE="$2"
TIMEOUT_SECONDS="${TEST_SQL_WAIT_TIMEOUT:-180}"

if [ -z "$CONTAINER" ] || [ -z "$ENGINE" ]; then
  echo "usage: wait-for-sql.sh <container-name> <mysql|mariadb|cockroachdb>" >&2
  exit 2
fi

probe() {
  case "$ENGINE" in
    mysql)
      docker exec "$CONTAINER" mysqladmin ping -uroot -ppassword >/dev/null 2>&1
      ;;
    mariadb)
      # MariaDB 11 renamed the client binaries; the mysql* names are only
      # compatibility symlinks and are being dropped, so try the real one first.
      docker exec "$CONTAINER" mariadb-admin ping -uroot -ppassword >/dev/null 2>&1 ||
        docker exec "$CONTAINER" mysqladmin ping -uroot -ppassword >/dev/null 2>&1
      ;;
    cockroachdb)
      docker exec "$CONTAINER" ./cockroach sql --insecure -e 'SELECT 1' >/dev/null 2>&1
      ;;
    *)
      echo "unknown engine '$ENGINE' (known: mysql, mariadb, cockroachdb)" >&2
      exit 2
      ;;
  esac
}

started=$(date +%s)
deadline=$(( started + TIMEOUT_SECONDS ))

echo "waiting for $ENGINE in $CONTAINER (up to ${TIMEOUT_SECONDS}s)..."
while :; do
  # Fail fast when the container died mid-wait: an exited container otherwise
  # looks exactly like a slow boot until the whole timeout burns down.
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

  if probe; then
    echo "  $CONTAINER serving $ENGINE (after $(( $(date +%s) - started ))s)"
    exit 0
  fi

  if [ "$(date +%s)" -ge "$deadline" ]; then
    echo "  $CONTAINER never served $ENGINE within ${TIMEOUT_SECONDS}s" >&2
    docker logs --tail 20 "$CONTAINER" 2>&1 | sed 's/^/    /' >&2
    exit 1
  fi
  sleep 2
done
