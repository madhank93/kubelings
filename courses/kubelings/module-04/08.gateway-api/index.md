---
kind: lesson
title: 'Gateway API: routing with a role for everyone'
description: |
  Ingress crams infra and app concerns into one object; Gateway API splits
  them — GatewayClass for the implementation, Gateway for the listener owned
  by platform, HTTPRoute for the routes owned by app teams. The platform half
  is already deployed. Write the app team's HTTPRoute and attach it.
name: gateway-api
slug: gateway-api
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
      # Gateway API kinds are CRDs (see M7.5) — install the pinned standard set.
      # Needs internet from the control-plane node (same reach as image pulls).
      if ! kubectl get crd httproutes.gateway.networking.k8s.io >/dev/null 2>&1; then
        kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.1/standard-install.yaml \
          || { echo "init failed: cannot fetch Gateway API CRDs (internet required from cplane-01)"; exit 1; }
      fi
      kubectl -n "$NS" delete httproute catalog --ignore-not-found
      kubectl -n "$NS" delete gateway web --ignore-not-found
      kubectl delete gatewayclass kubelings --ignore-not-found
      kubectl -n "$NS" delete deploy catalog --ignore-not-found
      kubectl -n "$NS" delete svc catalog --ignore-not-found
      kubectl apply -f - <<'YAML'
      apiVersion: gateway.networking.k8s.io/v1
      kind: GatewayClass
      metadata:
        name: kubelings
      spec:
        controllerName: kubelings.dev/unmanaged
      YAML
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: gateway.networking.k8s.io/v1
      kind: Gateway
      metadata:
        name: web
      spec:
        gatewayClassName: kubelings
        listeners:
          - name: http
            protocol: HTTP
            port: 80
            allowedRoutes:
              namespaces:
                from: Same
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: catalog
      spec:
        replicas: 2
        selector:
          matchLabels: {app: catalog}
        template:
          metadata:
            labels: {app: catalog}
          spec:
            containers:
              - name: catalog
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: catalog
      spec:
        selector: {app: catalog}
        ports: [{port: 80, targetPort: 80}]
      YAML
      kubectl -n "$NS" rollout status deploy/catalog --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get httproute catalog >/dev/null 2>&1; then
        echo "not yet: no HTTPRoute named catalog in kubelings — that's the app team's half, and that's you"; exit 1
      fi
      parent=$(kubectl -n "$NS" get httproute catalog -o jsonpath='{.spec.parentRefs[0].name}')
      if [ "$parent" != "web" ]; then
        echo "not yet: the route's parentRefs must attach it to the 'web' Gateway (got '${parent:-none}')"; exit 1
      fi
      bref=$(kubectl -n "$NS" get httproute catalog -o jsonpath='{.spec.rules[0].backendRefs[0].name}')
      if [ "$bref" != "catalog" ]; then
        echo "not yet: the rule's backendRefs must point at the catalog Service (got '${bref:-none}')"; exit 1
      fi
      bport=$(kubectl -n "$NS" get httproute catalog -o jsonpath='{.spec.rules[0].backendRefs[0].port}')
      if [ "$bport" != "80" ]; then
        echo "not yet: backendRef port must be 80 — the Service's port, same name-chain rule as the Ingress lesson"; exit 1
      fi
      eps=$(kubectl -n "$NS" get endpoints catalog -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null | wc -w | tr -d ' ')
      if [ "${eps:-0}" -lt 1 ]; then
        echo "not yet: the catalog Service has no endpoints — the route ends in a blackhole"; exit 1
      fi
      echo "PASS — GatewayClass → Gateway → HTTPRoute → Service → endpoints. Same chain discipline as Ingress, with the ownership seams in the right places."
---
