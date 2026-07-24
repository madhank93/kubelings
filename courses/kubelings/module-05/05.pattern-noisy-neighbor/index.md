---
kind: lesson
title: 'Drill — the noisy neighbor'
description: |
  Synthetic drill of the fleet-wide classic: one workload declares no resources
  and feasts, its neighbor is CPU-throttled into 2-second latency. Nothing
  crashes; everything is slow. Diagnose with resource math, fix with honest
  declarations.
name: pattern-noisy-neighbor
slug: pattern-noisy-neighbor
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
      TARGET=$(kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
        | grep -v control-plane | head -1)
      kubectl apply -n "$NS" -f - <<YAML
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: media-encoder
      spec:
        replicas: 1
        selector:
          matchLabels: {app: media-encoder}
        template:
          metadata:
            labels: {app: media-encoder}
          spec:
            nodeSelector:
              kubernetes.io/hostname: $TARGET
            containers:
              - name: encoder
                image: busybox:1.36
                # The neighbor: busy-loop, ZERO resource declarations. The
                # scheduler thinks it's free; the node knows better.
                command: ["sh", "-c"]
                args:
                  - |
                    while true; do :; done
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: quotes-api
      spec:
        replicas: 1
        selector:
          matchLabels: {app: quotes-api}
        template:
          metadata:
            labels: {app: quotes-api}
          spec:
            nodeSelector:
              kubernetes.io/hostname: $TARGET
            containers:
              - name: api
                image: nginx:1.27-alpine
                # The victim: microscopic CPU limit — throttled at the first breath.
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {cpu: 20m, memory: 128Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/media-encoder --timeout=180s
      kubectl -n "$NS" rollout status deploy/quotes-api --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # Neighbor must declare honest CPU requests + a limit.
      req=$(kubectl -n "$NS" get deploy media-encoder \
        -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}' 2>/dev/null)
      lim=$(kubectl -n "$NS" get deploy media-encoder \
        -o jsonpath='{.spec.template.spec.containers[0].resources.limits.cpu}' 2>/dev/null)
      if [ -z "$req" ] || [ -z "$lim" ]; then
        echo "not yet: media-encoder still declares no cpu requests/limits — the scheduler still thinks it's free"; exit 1
      fi
      # Victim's limit must allow real work (>= 100m).
      vlim=$(kubectl -n "$NS" get deploy quotes-api \
        -o jsonpath='{.spec.template.spec.containers[0].resources.limits.cpu}' 2>/dev/null)
      case "$vlim" in
        ""|"10m"|"20m"|"50m") echo "not yet: quotes-api cpu limit is '${vlim:-none}' — still throttled into molasses"; exit 1;;
      esac
      for d in media-encoder quotes-api; do
        avail=$(kubectl -n "$NS" get deploy "$d" -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
        if [ "${avail:-0}" -lt 1 ]; then echo "not yet: $d not Available"; exit 1; fi
      done
      echo "PASS — neighbor declared, victim unthrottled. Slow-not-down is a resources bug until proven otherwise."
---
