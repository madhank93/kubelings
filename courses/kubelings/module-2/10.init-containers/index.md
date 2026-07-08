---
kind: lesson
title: 'Stuck at Init: the gate that never opens'
description: |
  The `reports` pod has been Init:0/1 for twenty minutes. Its init container is
  waiting for configuration that will never arrive — it's checking the wrong
  name. Learn how init containers gate pod startup, and how to debug the gate.
name: init-containers
slug: init-containers
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
      # The config the init container SHOULD find:
      kubectl -n "$NS" create configmap reports-config \
        --from-literal=db_host=warehouse.internal \
        --dry-run=client -o yaml | kubectl apply -f -
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: reports
      spec:
        replicas: 1
        selector:
          matchLabels: {app: reports}
        template:
          metadata:
            labels: {app: reports}
          spec:
            initContainers:
              - name: wait-for-config
                image: busybox:1.36
                command: ["sh", "-c"]
                # BUG: polls 'report-config' — the ConfigMap is 'reports-config'.
                args:
                  - |
                    until test -f /cfg/db_host; do
                      echo "waiting for config volume..."; sleep 3
                    done
                    echo "config found, releasing the gate"
                volumeMounts:
                  - {name: cfg, mountPath: /cfg}
            containers:
              - name: reports
                image: busybox:1.36
                command: ["sh", "-c", "echo reports service up; while true; do sleep 5; done"]
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
            volumes:
              - name: cfg
                configMap:
                  name: report-config
                  optional: true
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      avail=$(kubectl -n "$NS" get deploy reports -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ "${avail:-0}" -lt 1 ]; then
        phase=$(kubectl -n "$NS" get pods -l app=reports -o jsonpath='{.items[0].status.phase}' 2>/dev/null)
        echo "not yet: reports not Available (pod phase: ${phase:-none}) — what is init waiting for?"; exit 1
      fi
      echo "PASS — the gate opened: init found its config and handed off to the app container."
---
