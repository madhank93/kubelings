---
kind: lesson
title: 'Incident file — the cluster that ran out of IP addresses (loveholidays)'
description: |
  Read-through of loveholidays' cited GKE incident plus its cousins (Neon,
  Veracode): pods stuck Pending, the autoscaler refusing to add nodes, and a
  subnet that was "obviously big enough" — until you count the way Kubernetes
  counts. No tasks; the arithmetic is done at cluster-creation time and cannot be
  patched later, so this lesson trains the sizing instinct.
name: incident-ip-exhaustion
slug: incident-ip-exhaustion
source: https://deploy.live/blog/when-gke-ran-out-of-ip-addresses/
createdAt: "2026-07-27"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 120
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      kubectl -n "$NS" delete configmap quick-check --ignore-not-found
  quick_check:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get configmap quick-check >/dev/null 2>&1; then
        echo "not yet: no ConfigMap 'quick-check' in namespace '$NS'"; exit 1
      fi
      wrong=""
      for kv in q1=c q2=a q3=b; do
        k=${kv%%=*}
        got=$(kubectl -n "$NS" get configmap quick-check -o jsonpath="{.data.$k}" | tr -d ' )' | tr 'A-Z' 'a-z')
        [ "$got" = "${kv#*=}" ] || wrong="$wrong $k"
      done
      if [ -n "$wrong" ]; then
        echo "not yet: re-read the lesson and try again:$wrong"; exit 1
      fi
      echo "PASS — quick check complete."
---
