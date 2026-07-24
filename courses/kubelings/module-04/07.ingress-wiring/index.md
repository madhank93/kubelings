---
kind: lesson
title: 'Ingress: three names that must agree'
description: |
  The new Ingress went out and the site 404s. An Ingress is pure wiring — host
  to path to service name to service port to pod — and every hop is a name
  that must match something real. Two of these names don't. Find them the way
  you found the selector mismatch: follow the chain.
name: ingress-wiring
slug: ingress-wiring
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
      kubectl -n "$NS" delete deploy storefront --ignore-not-found
      kubectl -n "$NS" delete svc storefront --ignore-not-found
      kubectl -n "$NS" delete ingress shop --ignore-not-found
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: storefront
      spec:
        replicas: 2
        selector:
          matchLabels: {app: storefront}
        template:
          metadata:
            labels: {app: storefront}
          spec:
            containers:
              - name: web
                image: nginx:1.27-alpine
                ports:
                  - name: http
                    containerPort: 80
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: storefront
      spec:
        selector: {app: storefront}
        ports:
          - name: web
            port: 8080
            targetPort: http
      ---
      # BUG(s): backend service name has a typo, and the port doesn't exist on
      # the service anyway. Every hop in the ingress chain is a name.
      apiVersion: networking.k8s.io/v1
      kind: Ingress
      metadata:
        name: shop
      spec:
        rules:
          - host: shop.example.com
            http:
              paths:
                - path: /
                  pathType: Prefix
                  backend:
                    service:
                      name: store-front
                      port:
                        number: 80
      YAML
      kubectl -n "$NS" rollout status deploy/storefront --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      svc=$(kubectl -n "$NS" get ingress shop -o jsonpath='{.spec.rules[0].http.paths[0].backend.service.name}' 2>/dev/null)
      if [ -z "$svc" ]; then
        echo "not yet: ingress 'shop' has no backend service — check the rule's backend"; exit 1
      fi
      if ! kubectl -n "$NS" get svc "$svc" >/dev/null 2>&1; then
        echo "not yet: ingress points at service '$svc', which doesn't exist — first name in the chain is broken"; exit 1
      fi
      pnum=$(kubectl -n "$NS" get ingress shop -o jsonpath='{.spec.rules[0].http.paths[0].backend.service.port.number}' 2>/dev/null)
      pname=$(kubectl -n "$NS" get ingress shop -o jsonpath='{.spec.rules[0].http.paths[0].backend.service.port.name}' 2>/dev/null)
      ok=""
      if [ -n "$pnum" ] && kubectl -n "$NS" get svc "$svc" -o jsonpath='{.spec.ports[*].port}' | tr ' ' '\n' | grep -qx "$pnum"; then ok=1; fi
      if [ -n "$pname" ] && kubectl -n "$NS" get svc "$svc" -o jsonpath='{.spec.ports[*].name}' | tr ' ' '\n' | grep -qx "$pname"; then ok=1; fi
      if [ -z "$ok" ]; then
        echo "not yet: ingress backend port (${pname:-$pnum}) doesn't exist on service '$svc' — second name in the chain is broken"; exit 1
      fi
      eps=$(kubectl -n "$NS" get endpoints "$svc" -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null | wc -w | tr -d ' ')
      if [ "${eps:-0}" -lt 1 ]; then
        echo "not yet: service '$svc' has no endpoints — the chain is wired but ends in a blackhole (selector? targetPort?)"; exit 1
      fi
      echo "PASS — host → path → service → port → endpoints, every name agreeing. That chain IS the Ingress."
---
