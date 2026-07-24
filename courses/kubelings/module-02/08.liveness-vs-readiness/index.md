---
kind: lesson
title: 'The liveness probe that kills healthy pods'
description: |
  Someone "added health checks" to the payments service and now it restarts
  forever — the liveness probe checks the wrong port, so the kubelet executes a
  perfectly healthy container every few seconds. Learn what liveness and
  readiness actually promise, and when each one should fail.
name: liveness-vs-readiness
slug: liveness-vs-readiness
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
        name: payments
      spec:
        replicas: 2
        selector:
          matchLabels: {app: payments}
        template:
          metadata:
            labels: {app: payments}
          spec:
            containers:
              - name: payments
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                # BUG: app serves on 80; probe checks 8080. Healthy pods get executed.
                livenessProbe:
                  httpGet: {path: /, port: 8080}
                  initialDelaySeconds: 5
                  periodSeconds: 5
                  failureThreshold: 2
                readinessProbe:
                  httpGet: {path: /, port: 80}
                  periodSeconds: 5
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      desired=$(kubectl -n "$NS" get deploy payments -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy payments -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: payments not Available (${avail:-0}/${desired:-?})"; exit 1
      fi
      # Liveness probe must exist and point at a real port.
      lport=$(kubectl -n "$NS" get deploy payments \
        -o jsonpath='{.spec.template.spec.containers[0].livenessProbe.httpGet.port}' 2>/dev/null)
      if [ -z "$lport" ]; then
        echo "not yet: keep a livenessProbe — fix it, don't delete it"; exit 1
      fi
      if [ "$lport" != "80" ]; then
        echo "not yet: livenessProbe still checks port $lport, the app serves on 80"; exit 1
      fi
      # Pods must have settled (kubelet stopped killing them).
      newest=$(kubectl -n "$NS" get pods -l app=payments --sort-by=.metadata.creationTimestamp \
        -o jsonpath='{.items[-1:].status.containerStatuses[0].restartCount}' 2>/dev/null)
      if [ "${newest:-0}" -gt 1 ]; then
        echo "not yet: pods are still being restarted by the kubelet (restarts=$newest)"; exit 1
      fi
      echo "PASS — probes now tell the truth: liveness=restart me, readiness=don't route to me."
---
