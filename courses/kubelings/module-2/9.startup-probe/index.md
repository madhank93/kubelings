---
kind: lesson
title: 'Slow starter vs impatient liveness'
description: |
  A legacy service needs ~40 seconds to warm up. Its liveness probe gives it 10.
  The pod is killed mid-boot, restarts, gets killed again — it will literally
  never finish starting. Fix it the modern way: a startupProbe that holds
  liveness fire until boot completes.
name: startup-probe
slug: startup-probe
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
        name: legacy-erp
      spec:
        replicas: 1
        selector:
          matchLabels: {app: legacy-erp}
        template:
          metadata:
            labels: {app: legacy-erp}
          spec:
            containers:
              - name: legacy-erp
                image: busybox:1.36
                command: ["sh", "-c"]
                args:
                  - |
                    echo "warming up (40s of caches, JIT, migrations)..."
                    sleep 40
                    echo "ready — serving on :8080"
                    touch /tmp/healthy
                    httpd -f -p 8080 -h /tmp
                # BUG: liveness fires from t=10s; app is alive at t=40s. It never gets there.
                livenessProbe:
                  exec:
                    command: ["test", "-f", "/tmp/healthy"]
                  initialDelaySeconds: 10
                  periodSeconds: 5
                  failureThreshold: 2
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      avail=$(kubectl -n "$NS" get deploy legacy-erp -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ "${avail:-0}" -lt 1 ]; then
        echo "not yet: legacy-erp not Available — still being killed mid-boot?"; exit 1
      fi
      if [ -z "$(kubectl -n "$NS" get deploy legacy-erp \
          -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}' 2>/dev/null)" ]; then
        echo "not yet: keep the livenessProbe — protect startup instead of removing it"; exit 1
      fi
      if [ -z "$(kubectl -n "$NS" get deploy legacy-erp \
          -o jsonpath='{.spec.template.spec.containers[0].startupProbe}' 2>/dev/null)" ]; then
        echo "not yet: add a startupProbe — that's the tool built for slow starters"; exit 1
      fi
      newest=$(kubectl -n "$NS" get pods -l app=legacy-erp --sort-by=.metadata.creationTimestamp \
        -o jsonpath='{.items[-1:].status.containerStatuses[0].restartCount}' 2>/dev/null)
      if [ "${newest:-0}" -gt 1 ]; then
        echo "not yet: newest pod has $newest restarts — startup still not protected"; exit 1
      fi
      echo "PASS — startupProbe holds liveness fire until boot completes. Slow ≠ dead."
---
