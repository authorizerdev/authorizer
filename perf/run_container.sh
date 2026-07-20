#!/usr/bin/env bash
# Stands up Authorizer + Postgres + Redis as containers on one Docker network
# for load testing — the setup this harness's README numbers came from.
# Not for production: dev JWT keys, an inline admin secret, and CSRF/rate-limit
# are relaxed for load testing (see perf/README.md's "gotchas" section).
#
# Usage: CPUS=8 MEM=4g ./perf/run_container.sh
set -euo pipefail

CPUS="${CPUS:-2}"
MEM="${MEM:-2g}"
ADMIN_SECRET="${ADMIN_SECRET:-$(openssl rand -hex 16)}"
NET=authorizer_perf_net
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

case "$(uname -m)" in
  arm64|aarch64) PLATFORM=linux/arm64 ;;
  *) PLATFORM=linux/amd64 ;;
esac

docker network create "$NET" >/dev/null 2>&1 || true
docker rm -vf authorizer_perf_pg authorizer_perf_redis authorizer_perf_app >/dev/null 2>&1 || true

docker run -d --name authorizer_perf_pg --network "$NET" \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=authorizer_perf postgres:16 >/dev/null
docker run -d --name authorizer_perf_redis --network "$NET" redis:7 >/dev/null
sleep 5

echo "Building authorizer:perf-local for $PLATFORM (matches this host — avoids emulation skewing the numbers)..."
docker build --platform "$PLATFORM" --build-arg VERSION=perf-local -t authorizer:perf-local "$REPO_ROOT" >/dev/null

docker run -d --name authorizer_perf_app \
  --network "$NET" --cpus="$CPUS" --memory="$MEM" \
  -p 8090:8080 -p 9092:9091 -p 8095:8081 \
  authorizer:perf-local \
  --admin-secret="$ADMIN_SECRET" \
  --database-type=postgres \
  --database-url="postgres://postgres:postgres@authorizer_perf_pg:5432/authorizer_perf?sslmode=disable" \
  --redis-url="redis://authorizer_perf_redis:6379" \
  --jwt-type=RS256 \
  --jwt-private-key="$(cat "$REPO_ROOT/perf/dev-jwt-private.pem")" \
  --jwt-public-key="$(cat "$REPO_ROOT/perf/dev-jwt-public.pem")" \
  --client-id=kbyuFDidLLm280LIwVFiazOqjO3ty8KH \
  --client-secret=60Op4HFM0I8ajz0WdiStAbziZ-VFQttXuxixHHs2R7r7-CW8GR79l-mmLqMhc-Sa \
  --allowed-origins=localhost:8090 \
  --rate-limit-rps=1000000 \
  --rate-limit-burst=1000000 >/dev/null

sleep 4
if ! curl -sf -o /dev/null http://localhost:8090/.well-known/openid-configuration; then
  echo "Server did not come up — check: docker logs authorizer_perf_app" >&2
  exit 1
fi

cat <<EOF
Ready: $CPUS vCPU / $MEM, http://localhost:8090 (gRPC :9092)
  ADMIN_SECRET=$ADMIN_SECRET

Next:
  k6 run -e BASE_URL=http://localhost:8090 -e ADMIN_SECRET=$ADMIN_SECRET -e TUPLES=200000 perf/k6/seed_fga.js
  k6 run -e BASE_URL=http://localhost:8090 -e VUS=20 -e DURATION=15s perf/k6/validate_jwt.js
  ./perf/stop_container.sh   # when done
EOF
