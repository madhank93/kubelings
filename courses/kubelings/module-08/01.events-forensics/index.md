---
kind: lesson
title: 'Events forensics: reconstruct the crime'
description: |
  A pod is gone and nobody knows why. No logs survive a deleted pod — but the
  cluster's events remember. Learn to read events as a timeline, the tool that
  turns "it just disappeared" into a precise cause.
name: events-forensics
slug: events-forensics
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
      # A deployment whose pod requests more memory than any node can give ->
      # FailedScheduling events, never runs. The learner must diagnose from
      # events and right-size it.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: analytics-job
      spec:
        replicas: 1
        selector:
          matchLabels: {app: analytics-job}
        template:
          metadata:
            labels: {app: analytics-job}
          spec:
            containers:
              - name: job
                image: nginx:1.27-alpine
                resources:
                  # BUG: 900Gi request — no node can satisfy it. Perpetual Pending.
                  requests: {cpu: 100m, memory: 900Gi}
                  limits: {memory: 900Gi}
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      avail=$(kubectl -n "$NS" get deploy analytics-job -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ "${avail:-0}" -lt 1 ]; then
        echo "not yet: analytics-job still not Available — what do its events say about scheduling?"; exit 1
      fi
      req=$(kubectl -n "$NS" get deploy analytics-job \
        -o jsonpath='{.spec.template.spec.containers[0].resources.requests.memory}' 2>/dev/null)
      case "$req" in
        *Gi) g=${req%Gi}; if [ "$g" -gt 8 ] 2>/dev/null; then
               echo "not yet: memory request $req still exceeds any node — read the FailedScheduling event and right-size it"; exit 1
             fi;;
      esac
      echo "PASS — you read the events, found the impossible request, right-sized it. Events are the cluster's black box recorder."
---
