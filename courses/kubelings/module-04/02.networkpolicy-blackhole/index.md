---
kind: lesson
title: 'NetworkPolicy blackhole: default-deny ate my traffic'
description: |
  Security rolled out a default-deny NetworkPolicy — correct move — but nobody
  wrote the allow rules that were supposed to ship with it. Learn how policies
  select, combine, and fail silently, and write the allow rule that lets exactly
  the right traffic through.
name: networkpolicy-blackhole
slug: networkpolicy-blackhole
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
        name: orders-api
      spec:
        replicas: 1
        selector:
          matchLabels: {app: orders-api}
        template:
          metadata:
            labels: {app: orders-api, tier: backend}
          spec:
            containers:
              - name: api
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: storefront
      spec:
        replicas: 1
        selector:
          matchLabels: {app: storefront}
        template:
          metadata:
            labels: {app: storefront, tier: frontend}
          spec:
            containers:
              - name: web
                image: busybox:1.36
                command: ["sh", "-c", "while true; do sleep 10; done"]
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: orders-api
      spec:
        selector: {app: orders-api}
        ports: [{port: 80, targetPort: 80}]
      ---
      # Security hardening PR, merged Friday. The allow rules were "a follow-up".
      apiVersion: networking.k8s.io/v1
      kind: NetworkPolicy
      metadata:
        name: default-deny-ingress
      spec:
        podSelector: {}
        policyTypes: [Ingress]
      YAML
      kubectl -n "$NS" rollout status deploy/orders-api --timeout=180s
      kubectl -n "$NS" rollout status deploy/storefront --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # default-deny must survive — the fix is an allow rule, not deleting the wall.
      if ! kubectl -n "$NS" get networkpolicy default-deny-ingress >/dev/null 2>&1; then
        echo "not yet: default-deny-ingress was deleted — keep the wall, add a door"; exit 1
      fi
      # An additional policy must allow ingress to orders-api from the frontend.
      allow_found=""
      for p in $(kubectl -n "$NS" get networkpolicy -o name | grep -v default-deny-ingress); do
        sel=$(kubectl -n "$NS" get "$p" -o jsonpath='{.spec.podSelector.matchLabels}' 2>/dev/null)
        ingress=$(kubectl -n "$NS" get "$p" -o jsonpath='{.spec.ingress}' 2>/dev/null)
        if grep -q 'orders-api\|backend' <<<"$sel" && [ -n "$ingress" ] && [ "$ingress" != "[]" ]; then
          allow_found=yes
        fi
      done
      if [ -z "$allow_found" ]; then
        echo "not yet: no NetworkPolicy allows ingress to orders-api — write the door"; exit 1
      fi
      # NOTE: kind's default CNI (kindnet) does not ENFORCE NetworkPolicy, so this
      # check validates the policy objects. On the Labs playground (enforcing CNI)
      # the same objects gate real traffic.
      if ! kubectl -n "$NS" run np-check --rm -i --restart=Never --image=busybox:1.36 \
          --labels=app=storefront,tier=frontend --timeout=60s \
          -- wget -q -O- --timeout=5 "http://orders-api.$NS.svc/" 2>/dev/null | grep -qi nginx; then
        echo "not yet: storefront-labeled pod cannot reach orders-api"; exit 1
      fi
      echo "PASS — wall intact, door installed: ingress to orders-api allowed only where you said so."
---
