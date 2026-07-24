---
kind: lesson
title: 'Connection refused: port vs targetPort'
description: |
  Endpoints populated, selector correct, pods healthy — and every request gets
  "connection refused". The Service forwards to a container port nobody listens
  on. Learn the port/targetPort/containerPort chain and the difference between
  refused and timeout.
name: broken-targetport
slug: broken-targetport
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
        name: search
      spec:
        replicas: 2
        selector:
          matchLabels: {app: search}
        template:
          metadata:
            labels: {app: search}
          spec:
            containers:
              - name: search
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: search
      spec:
        selector: {app: search}
        ports:
          # BUG: someone assumed the container serves on 8080. nginx: port 80.
          - port: 80
            targetPort: 8080
      YAML
      kubectl -n "$NS" rollout status deploy/search --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      tp=$(kubectl -n "$NS" get svc search -o jsonpath='{.spec.ports[0].targetPort}' 2>/dev/null)
      if [ "$tp" != "80" ]; then
        echo "not yet: Service still targets container port ${tp:-none} — who actually listens there?"; exit 1
      fi
      if ! kubectl -n "$NS" run tp-check --rm -i --restart=Never --image=busybox:1.36 \
          --timeout=60s -- wget -q -O- --timeout=5 "http://search.$NS.svc/" 2>/dev/null | grep -qi nginx; then
        echo "not yet: in-cluster request to search.$NS.svc still fails"; exit 1
      fi
      echo "PASS — port chain aligned: Service port → targetPort → containerPort → process. Refused means 'wrong door', not 'nobody home'."
---
