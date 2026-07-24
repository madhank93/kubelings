---
kind: lesson
title: 'Drill — the rolling update that deadlocks'
description: |
  Synthetic drill of a failure pattern reported across many production
  clusters: a rollout stuck for hours at "1 old replicas are pending
  termination". maxUnavailable: 0 means no old pod may die until a new one is
  Ready — and the new one can never be Ready, because its resource request
  fits on no node. Nothing crashes, nothing progresses. Break the deadlock.
name: pattern-rolling-update-deadlock
slug: pattern-rolling-update-deadlock
createdAt: "2026-07-13"
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
      # Revision 1: healthy and boring.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: inventory
        labels: {app: inventory}
      spec:
        replicas: 2
        strategy:
          type: RollingUpdate
          rollingUpdate:
            # Zero-downtime policy: never run below full capacity.
            maxUnavailable: 0
            maxSurge: 1
        selector:
          matchLabels: {app: inventory}
        template:
          metadata:
            labels: {app: inventory}
          spec:
            containers:
              - name: inventory
                image: busybox:1.36
                command: ["sh", "-c", "while true; do sleep 30; done"]
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/inventory --timeout=120s
      # Revision 2: v2 rollout with a copy-pasted resource block from a very
      # different machine. 64 CPUs fits on no node here -> Pending forever.
      kubectl -n "$NS" patch deploy inventory --type=json -p '[
        {"op": "add",     "path": "/spec/template/spec/containers/0/env",
         "value": [{"name": "VERSION", "value": "v2"}]},
        {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests",
         "value": {"cpu": "64", "memory": "16Mi"}}
      ]'
      # Do NOT wait for this rollout: being stuck IS the scenario.
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      v=$(kubectl -n "$NS" get deploy inventory -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="VERSION")].value}' 2>/dev/null)
      if [ "$v" != "v2" ]; then
        echo "not yet: the pod template no longer carries VERSION=v2 — 'rollout undo' cancels the release; ship v2, fixed"; exit 1
      fi
      if ! kubectl -n "$NS" rollout status deploy/inventory --timeout=60s >/dev/null 2>&1; then
        echo "not yet: rollout still stuck — why can the new pod not schedule? (kubectl describe the Pending pod)"; exit 1
      fi
      ready=$(kubectl -n "$NS" get deploy inventory -o jsonpath='{.status.readyReplicas}')
      if [ "${ready:-0}" -lt 2 ]; then
        echo "not yet: only ${ready:-0}/2 replicas Ready"; exit 1
      fi
      echo "PASS — v2 shipped without dropping below capacity. Deadlock broken at the cause, not the symptom."
---
