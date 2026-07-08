---
kind: lesson
title: 'Blue/green: the deploy you can undo in one second'
description: |
  Rolling updates replace pods gradually — great until a bad release must be
  yanked *now*. Blue/green runs old and new side by side and flips traffic with
  one selector change. The green stack is up, tested, and receiving zero
  traffic; flip the switch without dropping a request, and keep the way back.
name: blue-green-canary
slug: blue-green-canary
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
      kubectl -n "$NS" delete deploy shop-blue shop-green --ignore-not-found
      kubectl -n "$NS" delete svc shop --ignore-not-found
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: shop-blue
        labels: {app: shop, track: blue}
      spec:
        replicas: 2
        selector:
          matchLabels: {app: shop, track: blue}
        template:
          metadata:
            labels: {app: shop, track: blue, version: v1}
          spec:
            containers:
              - name: shop
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      ---
      # The v2 stack, built and ready — but scaled to zero and taking no traffic.
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: shop-green
        labels: {app: shop, track: green}
      spec:
        replicas: 0
        selector:
          matchLabels: {app: shop, track: green}
        template:
          metadata:
            labels: {app: shop, track: green, version: v2}
          spec:
            containers:
              - name: shop
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                readinessProbe:
                  httpGet: {path: /, port: 80}
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: shop
      spec:
        selector: {app: shop, track: blue}
        ports:
          - port: 80
            targetPort: 80
      YAML
      kubectl -n "$NS" rollout status deploy/shop-blue --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      track=$(kubectl -n "$NS" get svc shop -o jsonpath='{.spec.selector.track}' 2>/dev/null)
      if [ "$track" != "green" ]; then
        echo "not yet: service 'shop' still selects track=${track:-?} — traffic hasn't moved"; exit 1
      fi
      greads=$(kubectl -n "$NS" get deploy shop-green -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
      if [ "${greads:-0}" -lt 2 ]; then
        echo "not yet: shop-green has ${greads:-0}/2 ready pods — never flip traffic onto a stack that isn't ready"; exit 1
      fi
      eps=$(kubectl -n "$NS" get endpoints shop -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null | wc -w | tr -d ' ')
      if [ "${eps:-0}" -lt 2 ]; then
        echo "not yet: service 'shop' has $eps endpoints — selector and pod labels don't line up"; exit 1
      fi
      bl=$(kubectl -n "$NS" get deploy shop-blue -o jsonpath='{.spec.replicas}' 2>/dev/null)
      if [ -z "$bl" ] || [ "$bl" -lt 1 ]; then
        echo "not yet: shop-blue is gone or scaled to zero — keep the old stack alive until v2 has soaked; it IS your rollback"; exit 1
      fi
      echo "PASS — traffic on green, blue standing by for instant rollback. That flip was one field; that's the whole trick."
---
