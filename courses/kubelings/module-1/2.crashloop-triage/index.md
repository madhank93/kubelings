---
kind: lesson
title: 'CrashLoopBackOff: read the logs before you guess'
description: |
  The new `orders` service won't stay up — pods flash into Running, die within a
  second, and settle into CrashLoopBackOff. Nothing is wrong with the image. Learn
  the triage loop every later lesson builds on: describe → logs → fix → watch.
name: crashloop-triage
slug: crashloop-triage
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
      # The app refuses to boot without APP_MODE — and nobody set it.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: orders
      spec:
        replicas: 2
        selector:
          matchLabels: {app: orders}
        template:
          metadata:
            labels: {app: orders}
          spec:
            containers:
              - name: orders
                image: busybox:1.36
                command: ["sh", "-c"]
                args:
                  - |
                    if [ -z "${APP_MODE:-}" ]; then
                      echo "FATAL: required environment variable APP_MODE is not set" >&2
                      exit 1
                    fi
                    echo "orders service started in mode: $APP_MODE"
                    while true; do sleep 5; done
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      YAML
      # Don't wait for rollout — it crashloops by design.
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      desired=$(kubectl -n "$NS" get deploy orders -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy orders -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: orders Deployment not Available (${avail:-0}/${desired:-?}) — still crashlooping?"; exit 1
      fi
      # Pods must be stable, not merely mid-backoff.
      restarts=$(kubectl -n "$NS" get pods -l app=orders \
        -o jsonpath='{range .items[*]}{.status.containerStatuses[*].restartCount}{"\n"}{end}' 2>/dev/null \
        | awk '{s+=$1} END{print s+0}')
      waiting=$(kubectl -n "$NS" get pods -l app=orders \
        -o jsonpath='{range .items[*]}{.status.containerStatuses[*].state.waiting.reason}{"\n"}{end}' 2>/dev/null \
        | grep -c CrashLoopBackOff || true)
      if [ "${waiting:-0}" -gt 0 ]; then
        echo "not yet: some orders pods are still in CrashLoopBackOff"; exit 1
      fi
      echo "PASS — orders is running steadily (restarts total: ${restarts:-0}). You read the logs instead of guessing."
---
