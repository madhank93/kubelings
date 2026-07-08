---
kind: lesson
title: 'QoS classes: who gets killed first'
description: |
  Three tiers of workload, one node under memory pressure — and the kernel needs
  someone to die. Kubernetes decides the order via QoS classes computed from your
  requests/limits. Make the payment service Guaranteed, the batch job honest, and
  learn to read .status.qosClass before the OOM killer reads it for you.
name: qos-classes
slug: qos-classes
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
        name: pay-core
      spec:
        replicas: 1
        selector:
          matchLabels: {app: pay-core}
        template:
          metadata:
            labels: {app: pay-core}
          spec:
            containers:
              - name: pay-core
                image: nginx:1.27-alpine
                # Requests != limits -> only Burstable. Payments must be Guaranteed.
                resources:
                  requests: {cpu: 50m, memory: 64Mi}
                  limits: {cpu: 200m, memory: 128Mi}
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: batch-reindex
      spec:
        replicas: 1
        selector:
          matchLabels: {app: batch-reindex}
        template:
          metadata:
            labels: {app: batch-reindex}
          spec:
            containers:
              - name: batch-reindex
                image: busybox:1.36
                command: ["sh", "-c", "while true; do sleep 10; done"]
                # No requests/limits at all -> BestEffort AND invisible to the scheduler.
      YAML
      kubectl -n "$NS" rollout status deploy/pay-core --timeout=180s
      kubectl -n "$NS" rollout status deploy/batch-reindex --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      qos_pay=$(kubectl -n "$NS" get pods -l app=pay-core \
        -o jsonpath='{.items[0].status.qosClass}' 2>/dev/null)
      if [ "$qos_pay" != "Guaranteed" ]; then
        echo "not yet: pay-core QoS is '${qos_pay:-none}', needs Guaranteed (requests == limits, cpu AND memory)"; exit 1
      fi
      qos_batch=$(kubectl -n "$NS" get pods -l app=batch-reindex \
        -o jsonpath='{.items[0].status.qosClass}' 2>/dev/null)
      if [ "$qos_batch" = "BestEffort" ] || [ -z "$qos_batch" ]; then
        echo "not yet: batch-reindex is still BestEffort — give it honest requests (Burstable is fine)"; exit 1
      fi
      for d in pay-core batch-reindex; do
        avail=$(kubectl -n "$NS" get deploy "$d" -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
        if [ "${avail:-0}" -lt 1 ]; then echo "not yet: $d not Available"; exit 1; fi
      done
      echo "PASS — pay-core Guaranteed, batch honest. Under memory pressure the kernel now kills in the order you chose."
---
