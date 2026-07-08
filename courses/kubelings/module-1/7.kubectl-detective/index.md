---
kind: lesson
title: 'kubectl detective: find the broken one'
description: |
  Five microservices deployed, dashboards say "one of them is down", and that's
  all anyone knows. No names, no hints. Use get/describe/events to find the
  odd one out and bring it back. Speed comes from method, not luck.
name: kubectl-detective
slug: kubectl-detective
createdAt: "2026-07-07"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 240
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      for app in catalog checkout payments search; do
        kubectl -n "$NS" create deployment "$app" --image=nginx:1.27-alpine --replicas=1 \
          --dry-run=client -o yaml | kubectl apply -f -
      done
      # The culprit: scaled to zero during "maintenance", never scaled back.
      kubectl -n "$NS" create deployment recommendations --image=nginx:1.27-alpine --replicas=1 \
        --dry-run=client -o yaml | kubectl apply -f -
      kubectl -n "$NS" scale deploy/recommendations --replicas=0
      for app in catalog checkout payments search; do
        kubectl -n "$NS" rollout status deploy/"$app" --timeout=120s
      done
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      for app in catalog checkout payments search recommendations; do
        desired=$(kubectl -n "$NS" get deploy "$app" -o jsonpath='{.spec.replicas}' 2>/dev/null)
        avail=$(kubectl -n "$NS" get deploy "$app" -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
        if [ "${desired:-0}" -lt 1 ] || [ "${avail:-0}" -lt 1 ]; then
          echo "not yet: '$app' has no available replicas — is this the one?"; exit 1
        fi
      done
      echo "PASS — all five services alive. get → describe → events: the detective's loop."
---
