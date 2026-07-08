---
kind: lesson
title: 'Expose a Deployment: your first Service'
description: |
  The `web` frontend is running — two healthy nginx pods — but nothing in the
  cluster can reach it: there's no Service. Create one, watch endpoints populate,
  and learn how label selectors stitch traffic to pods.
name: expose-web
slug: expose-web
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
        name: web
      spec:
        replicas: 2
        selector:
          matchLabels: {app: web}
        template:
          metadata:
            labels: {app: web}
          spec:
            containers:
              - name: web
                image: nginx:1.27-alpine
                ports:
                  - containerPort: 80
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/web --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get svc web >/dev/null 2>&1; then
        echo "not yet: no Service named 'web' in namespace $NS"; exit 1
      fi
      port=$(kubectl -n "$NS" get svc web -o jsonpath='{.spec.ports[0].port}' 2>/dev/null)
      if [ "${port:-}" != "80" ]; then
        echo "not yet: Service 'web' must expose port 80 (got: ${port:-none})"; exit 1
      fi
      addrs=$(kubectl -n "$NS" get endpoints web \
        -o jsonpath='{range .subsets[*].addresses[*]}{.ip}{"\n"}{end}' 2>/dev/null | grep -c . || true)
      if [ "${addrs:-0}" -lt 2 ]; then
        echo "not yet: Service 'web' has ${addrs:-0} endpoint(s), expected 2 — selector matching the pods?"; exit 1
      fi
      # Prove traffic actually flows, from inside the cluster.
      if ! kubectl -n "$NS" run curl-check --rm -i --restart=Never --image=busybox:1.36 \
          --timeout=60s -- wget -q -O- --timeout=5 "http://web.$NS.svc/" 2>/dev/null | grep -qi nginx; then
        echo "not yet: in-cluster HTTP request to web.$NS.svc failed"; exit 1
      fi
      echo "PASS — 'web' Service is routing to both pods. Selectors → endpoints → traffic."
---
