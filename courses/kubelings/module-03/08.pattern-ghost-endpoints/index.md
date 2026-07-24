---
kind: lesson
title: 'Drill — ghost endpoints after scale-down'
description: |
  Synthetic drill of a failure pattern reported across many production
  clusters: every scale-down or rollout of a service throws a burst of 503s.
  terminationGracePeriodSeconds is 0 and there is no preStop hook — pods die
  the instant deletion starts, while their IPs are still in the endpoint
  list. Give termination the time the network needs.
name: pattern-ghost-endpoints
slug: pattern-ghost-endpoints
createdAt: "2026-07-13"
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
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: storefront
        labels: {app: storefront}
      spec:
        replicas: 2
        selector:
          matchLabels: {app: storefront}
        template:
          metadata:
            labels: {app: storefront}
          spec:
            # BUG: kill pods instantly — no time for endpoint removal to
            # propagate, no time for in-flight requests to drain.
            terminationGracePeriodSeconds: 0
            containers:
              - name: storefront
                image: busybox:1.36
                command: ["sh", "-c", "while true; do sleep 30; done"]
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: storefront
      spec:
        selector: {app: storefront}
        ports: [{port: 80, targetPort: 8080}]
      YAML
      kubectl -n "$NS" rollout status deploy/storefront --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      tgps=$(kubectl -n "$NS" get deploy storefront -o jsonpath='{.spec.template.spec.terminationGracePeriodSeconds}' 2>/dev/null)
      # empty means the field was removed -> Kubernetes default of 30s, fine.
      if [ -n "$tgps" ] && [ "$tgps" -lt 5 ]; then
        echo "not yet: terminationGracePeriodSeconds is $tgps — pods still die before the endpoint list catches up"; exit 1
      fi
      prestop=$(kubectl -n "$NS" get deploy storefront -o jsonpath='{.spec.template.spec.containers[0].lifecycle.preStop}' 2>/dev/null)
      if [ -z "$prestop" ]; then
        echo "not yet: no preStop hook — nothing holds the pod alive while its IP leaves the endpoint list"; exit 1
      fi
      if ! kubectl -n "$NS" rollout status deploy/storefront --timeout=120s >/dev/null 2>&1; then
        echo "not yet: storefront rollout not complete"; exit 1
      fi
      echo "PASS — termination now waits for the network: preStop drains, grace period covers it. No more 503 bursts on scale-down."
---
