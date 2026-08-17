#!/usr/bin/env bash
# Kubernetes workload-identity e2e: create a throwaway cluster, extract the facts
# a real cluster publishes, run the `k8s`-tagged Go suite against them, tear down.
#
# The point of using a REAL cluster is the addresses. Every existing test
# substitutes an httptest server, which replaces the one property that decides
# whether the feature works in production: a cluster publishes its issuer, JWKS
# and apiserver on private/loopback addresses, and validators.SafeHTTPClient
# rejects those unconditionally. Only a real cluster reproduces that.
#
# Usage:
#   make test-k8s                      # kind (default)
#   K8S_RUNTIME=k3d make test-k8s      # k3d
#   K8S_KEEP=1 make test-k8s           # leave the cluster up for debugging
set -euo pipefail

RUNTIME="${K8S_RUNTIME:-kind}"
CLUSTER="${K8S_CLUSTER_NAME:-authorizer-e2e}"
NAMESPACE="default"
SA_NAME="authorizer-workload"
AUDIENCE="${K8S_AUDIENCE:-https://authorizer.example.org}"
KEEP="${K8S_KEEP:-0}"

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }

for bin in kubectl go; do
  command -v "$bin" >/dev/null || { echo "missing required tool: $bin" >&2; exit 1; }
done
command -v "$RUNTIME" >/dev/null || { echo "missing cluster runtime: $RUNTIME" >&2; exit 1; }

created=0
cleanup() {
  local rc=$?
  if [ "$created" = "1" ] && [ "$KEEP" != "1" ]; then
    log "tearing down $RUNTIME cluster $CLUSTER"
    case "$RUNTIME" in
      kind) kind delete cluster --name "$CLUSTER" >/dev/null 2>&1 || true ;;
      k3d)  k3d cluster delete "$CLUSTER"        >/dev/null 2>&1 || true ;;
    esac
  elif [ "$created" = "1" ]; then
    log "K8S_KEEP=1 — leaving cluster $CLUSTER up"
  fi
  exit $rc
}
trap cleanup EXIT

cluster_exists() {
  case "$RUNTIME" in
    kind) kind get clusters 2>/dev/null | grep -qx "$CLUSTER" ;;
    k3d)  k3d cluster list -o json 2>/dev/null | grep -q "\"name\":\"$CLUSTER\"" ;;
  esac
}

if cluster_exists; then
  log "reusing existing $RUNTIME cluster $CLUSTER"
else
  log "creating $RUNTIME cluster $CLUSTER"
  case "$RUNTIME" in
    kind) kind create cluster --name "$CLUSTER" --wait 120s ;;
    k3d)  k3d cluster create "$CLUSTER" --wait ;;
  esac
  created=1
fi

case "$RUNTIME" in
  kind) kubectl config use-context "kind-$CLUSTER" >/dev/null ;;
  k3d)  kubectl config use-context "k3d-$CLUSTER"  >/dev/null ;;
esac

log "extracting cluster facts"
# Anonymous access to the discovery document is off by default on some
# distributions; fall back to the value the apiserver stamps into tokens.
DISCOVERY="$(kubectl get --raw /.well-known/openid-configuration 2>/dev/null || echo '')"
if [ -n "$DISCOVERY" ]; then
  K8S_ISSUER="$(printf '%s' "$DISCOVERY"   | python3 -c 'import json,sys;print(json.load(sys.stdin)["issuer"])')"
  K8S_JWKS_URI="$(printf '%s' "$DISCOVERY" | python3 -c 'import json,sys;print(json.load(sys.stdin)["jwks_uri"])')"
else
  echo "cluster does not serve /.well-known/openid-configuration anonymously" >&2
  exit 1
fi
K8S_APISERVER="$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')"

kubectl create serviceaccount "$SA_NAME" -n "$NAMESPACE" >/dev/null 2>&1 || true
K8S_SA_TOKEN="$(kubectl create token "$SA_NAME" -n "$NAMESPACE" --audience="$AUDIENCE" --duration=1h)"
K8S_SA_SUBJECT="system:serviceaccount:${NAMESPACE}:${SA_NAME}"

log "issuer=$K8S_ISSUER"
log "jwks_uri=$K8S_JWKS_URI"
log "apiserver=$K8S_APISERVER"
log "subject=$K8S_SA_SUBJECT"

export K8S_ISSUER K8S_JWKS_URI K8S_APISERVER K8S_SA_TOKEN K8S_SA_SUBJECT
export K8S_AUDIENCE="$AUDIENCE"
export TEST_DBS=sqlite

log "running the k8s-tagged suite"
go test -tags k8s -count=1 -p 1 -v -timeout 10m \
  -run 'TestK8s' ./internal/integration_tests/
