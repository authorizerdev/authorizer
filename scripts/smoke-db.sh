#!/bin/sh
# Run the release smoke's database subset (TestReleaseSmokeDB) against a real
# database in Docker: `make smoke-db DB=serenedb|surrealdb|sqlserver`.
#
# The container is always removed on exit, including when the tests fail, and
# the script exits with the tests' status. Host port and container name are
# overridable (SMOKE_DB_PORT, SMOKE_DB_CONTAINER) so the run does not collide
# with a `make test-<db>` container on the default port.
set -eu

DB="${1:-}"
CONTAINER="${SMOKE_DB_CONTAINER:-authorizer_smoke_$DB}"
SQLSERVER_PASSWORD="${SQLSERVER_PASSWORD:-Str0ng!Passw0rd}"

case "$DB" in
  serenedb)
    PORT="${SMOKE_DB_PORT:-17890}"
    RUN="-p $PORT:7890 -e POSTGRES_PASSWORD=postgres ${SERENEDB_IMAGE:-serenedb/serenedb:26.11.2}"
    ARGS="--database-type=serenedb --database-url=postgres://postgres:postgres@127.0.0.1:$PORT/postgres"
    ;;
  surrealdb)
    PORT="${SMOKE_DB_PORT:-18002}"
    RUN="-p $PORT:8000 ${SURREALDB_IMAGE:-surrealdb/surrealdb:v4.0.1} start --user root --pass root rocksdb:/tmp/authorizer.db"
    ARGS="--database-type=surrealdb --database-url=http://127.0.0.1:$PORT --database-username=root --database-password=root --database-name=authorizer --surrealdb-namespace=authorizer_smoke"
    ;;
  sqlserver)
    PORT="${SMOKE_DB_PORT:-11433}"
    RUN="-p $PORT:1433 -e ACCEPT_EULA=Y -e MSSQL_SA_PASSWORD=$SQLSERVER_PASSWORD ${SQLSERVER_IMAGE:-mcr.microsoft.com/mssql/server:2022-latest}"
    ARGS="--database-type=sqlserver --database-url=sqlserver://sa:$SQLSERVER_PASSWORD@127.0.0.1:$PORT?database=authorizer_smoke"
    ;;
  *)
    echo "usage: make smoke-db DB=serenedb|surrealdb|sqlserver" >&2
    exit 2
    ;;
esac

docker rm -vf "$CONTAINER" >/dev/null 2>&1 || true
trap 'docker rm -vf "$CONTAINER" >/dev/null 2>&1 || true' EXIT
# shellcheck disable=SC2086 # RUN is a word list on purpose.
docker run -d --name "$CONTAINER" $RUN >/dev/null

SQLSERVER_PASSWORD="$SQLSERVER_PASSWORD" sh "$(dirname "$0")/wait-for-sql.sh" "$CONTAINER" "$DB"
if [ "$DB" = sqlserver ]; then
  docker exec "$CONTAINER" /opt/mssql-tools18/bin/sqlcmd -C -S localhost -U sa \
    -P "$SQLSERVER_PASSWORD" -Q "CREATE DATABASE authorizer_smoke"
fi

SMOKE_DB_ARGS="$ARGS" go test -tags smoke -count=1 -v -timeout 10m -run '^TestReleaseSmokeDB$' ./internal/e2e/
