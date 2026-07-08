---
kind: lesson
title: 'ContainerCreating forever: the Secret that isn''t'
description: |
  The `gateway` pod has been ContainerCreating for ten minutes. Its TLS Secret
  volume references a Secret that doesn't exist under that name. Learn how Secret
  volumes gate container creation, and the difference between a missing Secret
  and a missing key.
name: secret-not-mounted
slug: secret-not-mounted
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
      # The real Secret (as the cert-rotation job names it):
      kubectl -n "$NS" create secret generic gateway-tls-cert \
        --from-literal=tls.crt="FAKE-CERT-FOR-LESSON" \
        --from-literal=tls.key="FAKE-KEY-FOR-LESSON" \
        --dry-run=client -o yaml | kubectl apply -f -
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: gateway
      spec:
        replicas: 1
        selector:
          matchLabels: {app: gateway}
        template:
          metadata:
            labels: {app: gateway}
          spec:
            containers:
              - name: gateway
                image: nginx:1.27-alpine
                volumeMounts:
                  - {name: tls, mountPath: /etc/tls, readOnly: true}
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
            volumes:
              - name: tls
                secret:
                  # BUG: rotation job writes 'gateway-tls-cert'; this asks for 'gateway-tls'.
                  secretName: gateway-tls
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      avail=$(kubectl -n "$NS" get deploy gateway -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ "${avail:-0}" -lt 1 ]; then
        echo "not yet: gateway not Available — what do the pod's events say about its volumes?"; exit 1
      fi
      pod=$(kubectl -n "$NS" get pods -l app=gateway -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
      if ! kubectl -n "$NS" exec "$pod" -- test -f /etc/tls/tls.crt 2>/dev/null; then
        echo "not yet: /etc/tls/tls.crt is not mounted inside the gateway container"; exit 1
      fi
      echo "PASS — Secret volume mounted. Missing Secret = pod gated at creation; that's your early warning."
---
