---
kind: lesson
title: 'The Service that routes to nothing'
description: |
  The `api` Service exists, DNS resolves, the pods are healthy — and every
  request times out. Its endpoints list is empty: the selector matches zero pods.
  One character is wrong somewhere. Find it.
name: selector-mismatch
slug: selector-mismatch
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
        name: api
      spec:
        replicas: 2
        selector:
          matchLabels: {app: api}
        template:
          metadata:
            labels: {app: api}
          spec:
            containers:
              - name: api
                image: nginx:1.27-alpine
                ports:
                  - containerPort: 80
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: api
      spec:
        # BUG: pods are labeled app=api, this selects app=api-server -> 0 endpoints.
        selector:
          app: api-server
        ports:
          - port: 80
            targetPort: 80
      YAML
      kubectl -n "$NS" rollout status deploy/api --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      addrs=$(kubectl -n "$NS" get endpoints api \
        -o jsonpath='{range .subsets[*].addresses[*]}{.ip}{"\n"}{end}' 2>/dev/null | grep -c . || true)
      if [ "${addrs:-0}" -lt 2 ]; then
        echo "not yet: Service 'api' has ${addrs:-0} endpoint(s), expected 2 — compare selector vs pod labels"; exit 1
      fi
      if ! kubectl -n "$NS" run curl-check-api --rm -i --restart=Never --image=busybox:1.36 \
          --timeout=60s -- wget -q -O- --timeout=5 "http://api.$NS.svc/" 2>/dev/null | grep -qi nginx; then
        echo "not yet: in-cluster HTTP request to api.$NS.svc still fails"; exit 1
      fi
      echo "PASS — selector matches, endpoints populated, traffic flows. Empty endpoints will never fool you again."
---
