#!/usr/bin/env bash
# Tears down everything perf/run_container.sh started.
set -euo pipefail
docker rm -vf authorizer_perf_app authorizer_perf_pg authorizer_perf_redis >/dev/null 2>&1 || true
docker network rm authorizer_perf_net >/dev/null 2>&1 || true
echo "Stopped."
