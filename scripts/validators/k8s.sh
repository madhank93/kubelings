#!/usr/bin/env bash
# Kubelings Kubernetes validator helpers — OPTIONAL local reference library.
#
# Published challenges are SELF-CONTAINED: their verify tasks inline plain kubectl
# checks and do NOT source this file (no network/repo dependency at runtime). Keep
# this lib for local testing and as the canonical shape of each check — when you
# inline a check into a challenge, mirror the logic here.
#
#   source scripts/validators/k8s.sh   # local use only
#
# Every function returns 0 (pass) / non-zero (fail) and is quiet. Do not `set -e`
# before sourcing: these helpers rely on non-zero returns as control flow.

# ── Pods ────────────────────────────────────────────────────────────────────
k8s_pod_running() {
  # k8s_pod_running <pod> [ns]
  local name=$1 ns=${2:-default}
  [ "$(kubectl get pod "$name" -n "$ns" -o jsonpath='{.status.phase}' 2>/dev/null)" = "Running" ]
}

k8s_pod_count_running() {
  # k8s_pod_count_running <selector> <ns> <min-expected>
  local selector=$1 ns=$2 expected=$3 actual
  actual=$(kubectl get pods -n "$ns" -l "$selector" \
    --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l | tr -d ' ')
  [ "${actual:-0}" -ge "$expected" ]
}

k8s_pod_restarts_below() {
  # k8s_pod_restarts_below <selector> <ns> <max-restarts>
  local selector=$1 ns=$2 max=$3 total
  total=$(kubectl get pods -n "$ns" -l "$selector" \
    -o jsonpath='{range .items[*]}{.status.containerStatuses[*].restartCount}{"\n"}{end}' \
    2>/dev/null | awk '{s+=$1} END{print s+0}')
  [ "${total:-0}" -le "$max" ]
}

k8s_pod_has_limits() {
  # k8s_pod_has_limits <selector> <ns> [resource=memory]
  # True when every matching pod sets a limit for <resource> on its first container.
  local selector=$1 ns=$2 res=${3:-memory} vals
  vals=$(kubectl get pods -n "$ns" -l "$selector" \
    -o jsonpath="{range .items[*]}{.spec.containers[0].resources.limits.$res}{\"\n\"}{end}" \
    2>/dev/null)
  [ -n "$vals" ] && ! grep -qx "" <<<"$vals"
}

# ── Deployments / rollouts ────────────────────────────────────────────────────
k8s_deployment_ready() {
  # k8s_deployment_ready <deploy> [ns] — available replicas >= desired
  local name=$1 ns=${2:-default} desired available
  desired=$(kubectl get deploy "$name" -n "$ns" -o jsonpath='{.spec.replicas}' 2>/dev/null)
  available=$(kubectl get deploy "$name" -n "$ns" -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
  [ -n "$desired" ] && [ "${available:-0}" -ge "$desired" ]
}

k8s_rollout_complete() {
  # k8s_rollout_complete <kind/name> <ns> [timeout=60s]
  local target=$1 ns=$2 timeout=${3:-60s}
  kubectl -n "$ns" rollout status "$target" --timeout="$timeout" >/dev/null 2>&1
}

# ── Autoscaling ───────────────────────────────────────────────────────────────
k8s_hpa_scaled() {
  # k8s_hpa_scaled <hpa> <ns> <min-current-replicas>
  local name=$1 ns=$2 min=$3 cur
  cur=$(kubectl get hpa "$name" -n "$ns" -o jsonpath='{.status.currentReplicas}' 2>/dev/null)
  [ "${cur:-0}" -ge "$min" ]
}

# ── Generic existence ─────────────────────────────────────────────────────────
k8s_resource_exists() {
  # k8s_resource_exists <kind> <name> [ns]
  local kind=$1 name=$2 ns=${3:-default}
  kubectl get "$kind" "$name" -n "$ns" >/dev/null 2>&1
}
