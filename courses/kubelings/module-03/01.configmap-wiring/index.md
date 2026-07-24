---
kind: lesson
title: 'CreateContainerConfigError: the key that isn''t there'
description: |
  The `pricing` service won't start — not crashing, not pulling, just
  CreateContainerConfigError. It wants an env var from a ConfigMap key that
  doesn't exist. Learn how config wires into containers (env vs mounts) and why
  the two behave differently when config changes.
name: configmap-wiring
slug: configmap-wiring
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
      kubectl -n "$NS" create configmap pricing-config \
        --from-literal=currency=EUR --from-literal=tax_rate=0.19 \
        --dry-run=client -o yaml | kubectl apply -f -
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: pricing
      spec:
        replicas: 1
        selector:
          matchLabels: {app: pricing}
        template:
          metadata:
            labels: {app: pricing}
          spec:
            containers:
              - name: pricing
                image: busybox:1.36
                command: ["sh", "-c", "echo pricing up: currency=$CURRENCY tax=$TAX_RATE; while true; do sleep 5; done"]
                env:
                  - name: CURRENCY
                    valueFrom:
                      configMapKeyRef: {name: pricing-config, key: currency}
                  - name: TAX_RATE
                    valueFrom:
                      # BUG: the key is 'tax_rate'; this asks for 'taxRate'.
                      configMapKeyRef: {name: pricing-config, key: taxRate}
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      avail=$(kubectl -n "$NS" get deploy pricing -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ "${avail:-0}" -lt 1 ]; then
        reason=$(kubectl -n "$NS" get pods -l app=pricing \
          -o jsonpath='{.items[0].status.containerStatuses[0].state.waiting.reason}' 2>/dev/null)
        echo "not yet: pricing not Available (waiting: ${reason:-unknown}) — describe the pod"; exit 1
      fi
      pod=$(kubectl -n "$NS" get pods -l app=pricing -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
      tax=$(kubectl -n "$NS" exec "$pod" -- sh -c 'echo $TAX_RATE' 2>/dev/null)
      if [ "$tax" != "0.19" ]; then
        echo "not yet: TAX_RATE inside the container is '${tax:-empty}', expected 0.19"; exit 1
      fi
      echo "PASS — env wired to the real key. Remember: env is copied at start; mounts stay live."
---
