---
kind: lesson
title: 'Incident replay — 502s on every deploy (Ravelin''s endpoint secret)'
description: |
  Replay of Ravelin's cited write-up: every rolling update produced a burst of
  502s, because pod termination and endpoint removal run in PARALLEL, not in
  order. The fix is a deliberately "dumb" preStop sleep. Understand the race,
  then wire the fix.
name: incident-graceful-shutdown
slug: incident-graceful-shutdown
source: https://philpearl.github.io/post/k8s_ingress/
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
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: checkout-api
      spec:
        replicas: 2
        selector:
          matchLabels: {app: checkout-api}
        template:
          metadata:
            labels: {app: checkout-api}
          spec:
            # BUG(race): no preStop delay + tiny grace period. On every rollout,
            # pods die BEFORE all nodes/ingresses stop routing to them -> 502 burst.
            terminationGracePeriodSeconds: 1
            containers:
              - name: api
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: checkout-api
      spec:
        selector: {app: checkout-api}
        ports: [{port: 80, targetPort: 80}]
      YAML
      kubectl -n "$NS" rollout status deploy/checkout-api --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      pre=$(kubectl -n "$NS" get deploy checkout-api \
        -o jsonpath='{.spec.template.spec.containers[0].lifecycle.preStop}' 2>/dev/null)
      if [ -z "$pre" ]; then
        echo "not yet: no preStop hook — the pod still exits the instant SIGTERM lands, mid-route"; exit 1
      fi
      if ! grep -q 'sleep' <<<"$pre"; then
        echo "not yet: preStop exists but doesn't wait — the standard fix is a short sleep"; exit 1
      fi
      grace=$(kubectl -n "$NS" get deploy checkout-api \
        -o jsonpath='{.spec.template.spec.terminationGracePeriodSeconds}' 2>/dev/null)
      if [ "${grace:-0}" -lt 10 ]; then
        echo "not yet: terminationGracePeriodSeconds=${grace:-1} — the grace period must cover preStop + drain"; exit 1
      fi
      avail=$(kubectl -n "$NS" get deploy checkout-api -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ "${avail:-0}" -lt 2 ]; then
        echo "not yet: checkout-api not fully Available after the change"; exit 1
      fi
      echo "PASS — termination now outlasts endpoint propagation. Deploys stop being outages."
---
