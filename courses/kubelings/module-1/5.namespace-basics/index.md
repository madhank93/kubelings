---
kind: lesson
title: 'Namespaces: walls, names, and crossing them'
description: |
  The platform team wants a `staging` environment isolated from everything else,
  and the app config copied into it. Learn what namespaces isolate (names, RBAC,
  quotas) and what they don't (network, nodes) — and how DNS crosses the wall.
name: namespace-basics
slug: namespace-basics
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
      kubectl delete namespace staging --ignore-not-found --wait=true
      kubectl -n "$NS" create configmap app-config \
        --from-literal=env=production --from-literal=region=eu-central-1 \
        --dry-run=client -o yaml | kubectl apply -f -
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: config-api
      spec:
        replicas: 1
        selector:
          matchLabels: {app: config-api}
        template:
          metadata:
            labels: {app: config-api}
          spec:
            containers:
              - name: web
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      YAML
      kubectl -n "$NS" expose deploy/config-api --port=80 --name=config-api \
        --dry-run=client -o yaml | kubectl apply -n "$NS" -f -
      kubectl -n "$NS" rollout status deploy/config-api --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl get namespace staging >/dev/null 2>&1; then
        echo "not yet: namespace 'staging' does not exist"; exit 1
      fi
      env_val=$(kubectl -n staging get configmap app-config -o jsonpath='{.data.env}' 2>/dev/null)
      if [ -z "$env_val" ]; then
        echo "not yet: configmap 'app-config' not found in namespace 'staging' — copy it over"; exit 1
      fi
      # Cross-namespace reachability: staging pod must reach the service in kubelings by FQDN.
      if ! kubectl -n staging run xns-check --rm -i --restart=Never --image=busybox:1.36 \
          --timeout=60s -- wget -q -O- --timeout=5 "http://config-api.$NS.svc.cluster.local/" 2>/dev/null | grep -qi nginx; then
        echo "not yet: a pod in 'staging' cannot reach config-api.$NS.svc.cluster.local"; exit 1
      fi
      echo "PASS — staging exists, config copied, and you crossed the namespace wall by FQDN."
---
