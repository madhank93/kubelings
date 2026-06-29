#!/usr/bin/env bash
# Thin local runner for kubelings exercises on a kind cluster.
# Usage: run-local.sh <exercise-dir> [up|init|verify|reset|solve|down]
set -euo pipefail

EX="${1:?usage: run-local.sh <exercise-dir> [up|init|verify|reset|solve|down]}"
CMD="${2:-verify}"
CLUSTER="${KUBELINGS_CLUSTER:-kubelings}"
NS="${KUBELINGS_NS:-kubelings}"
export KUBELINGS_NS="$NS"

[ -d "$EX" ] || { echo "no such exercise dir: $EX"; exit 2; }

case "$CMD" in
  up)
    if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
      kind create cluster --name "$CLUSTER"
      # Pre-pull the image into the node so exercises start fast / offline.
      docker pull nginx:1.27-alpine >/dev/null 2>&1 || true
      kind load docker-image nginx:1.27-alpine --name "$CLUSTER" >/dev/null 2>&1 || true
    fi
    kubectl config use-context "kind-$CLUSTER" >/dev/null
    echo "Cluster '$CLUSTER' ready."
    ;;
  init)   bash "$EX/init.sh" ;;
  verify) bash "$EX/verify.sh" ;;
  solve)
    kubectl apply -n "$NS" -f "$EX/solution.yaml"
    bash "$EX/verify.sh"
    ;;
  reset)
    kubectl delete namespace "$NS" --ignore-not-found --wait=true
    bash "$EX/init.sh"
    ;;
  down)
    kind delete cluster --name "$CLUSTER"
    ;;
  *)
    echo "unknown command: $CMD (use up|init|verify|reset|solve|down)"; exit 2 ;;
esac
