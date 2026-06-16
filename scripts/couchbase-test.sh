#!/bin/sh

set -eu

wait_for_admin() {
	i=0
	while [ "$i" -lt 90 ]; do
		if curl -sf http://127.0.0.1:8091/pools >/dev/null 2>&1; then
			return 0
		fi
		i=$((i + 1))
		sleep 2
	done
	echo "Couchbase admin UI not ready on 127.0.0.1:8091" >&2
	return 1
}

wait_for_healthy() {
	i=0
	while [ "$i" -lt 90 ]; do
		if curl -sf -u Administrator:password http://127.0.0.1:8091/pools/default | grep -q '"status":"healthy"'; then
			return 0
		fi
		i=$((i + 1))
		sleep 2
	done
	echo "Couchbase cluster did not reach healthy status" >&2
	return 1
}

wait_for_admin

# Setup services
curl -sf http://127.0.0.1:8091/node/controller/setupServices -d services=kv%2Cn1ql%2Cindex

# Setup credentials
curl -sf http://127.0.0.1:8091/settings/web -d port=8091 -d username=Administrator -d password=password

# Setup Memory Optimized Indexes
curl -sf -u Administrator:password -X POST http://127.0.0.1:8091/settings/indexes -d 'storageMode=memory_optimized'

wait_for_healthy

echo "Type: ${TYPE:-}"

if [ "${TYPE:-}" = "WORKER" ]; then
	echo "Sleeping ..."
	sleep 15

	IP=$(hostname -I | cut -d ' ' -f1)
	echo "IP: $IP"

	echo "Auto Rebalance: ${AUTO_REBALANCE:-}"
	if [ "${AUTO_REBALANCE:-}" = "true" ]; then
		couchbase-cli rebalance --cluster="${COUCHBASE_MASTER}:8091" --user=Administrator --password=password --server-add="$IP" --server-add-username=Administrator --server-add-password=password
	else
		couchbase-cli server-add --cluster="${COUCHBASE_MASTER}:8091" --user=Administrator --password=password --server-add="$IP" --server-add-username=Administrator --server-add-password=password
	fi
fi
