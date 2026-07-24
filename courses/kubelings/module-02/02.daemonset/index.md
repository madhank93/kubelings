---
kind: lesson
title: Build a Node-Level Log Collector DaemonSet
description: |
  Ops needs a log shipper running on every node — exactly one pod per node, now
  and on any node added later. Build a DaemonSet that satisfies this and confirm
  it lands a Ready pod on each node.
name: daemonset
slug: daemonset
createdAt: "2026-06-30"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 180
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      kubectl wait --for=condition=Ready nodes --all --timeout=120s || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      kubectl -n "$NS" get daemonset node-logger >/dev/null 2>&1 || {
        echo "not yet: no DaemonSet 'node-logger' in $NS"; exit 1; }
      desired=$(kubectl -n "$NS" get ds node-logger -o jsonpath='{.status.desiredNumberScheduled}')
      ready=$(kubectl -n "$NS" get ds node-logger -o jsonpath='{.status.numberReady}')
      if [ "${desired:-0}" -lt 1 ]; then
        echo "not yet: DaemonSet desires 0 pods — is it scheduled on nodes?"; exit 1
      fi
      if [ "${ready:-0}" -ne "${desired:-0}" ]; then
        echo "not yet: $ready/$desired DaemonSet pods Ready"; exit 1
      fi
      echo "PASS — node-logger is Ready on all $desired node(s)."
---
