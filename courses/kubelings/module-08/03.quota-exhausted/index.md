---
kind: lesson
title: 'Deploy blocked: the quota nobody mentioned'
description: |
  A scale-up half-worked: 2 of 5 pods created, then FailedCreate, and the
  Deployment is stuck. A ResourceQuota is out of room — and the error is on the
  ReplicaSet, not the pods. Learn to read quota, and the layer where this class
  of failure hides.
name: quota-exhausted
slug: quota-exhausted
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
      kubectl -n "$NS" delete resourcequota kubelings-quota --ignore-not-found
      # Quota allows ~2 small pods' worth of memory; deployment wants 5.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: ResourceQuota
      metadata:
        name: kubelings-quota
      spec:
        hard:
          requests.memory: "160Mi"
          limits.memory: "320Mi"
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: workers
      spec:
        replicas: 5
        selector:
          matchLabels: {app: workers}
        template:
          metadata:
            labels: {app: workers}
          spec:
            containers:
              - name: worker
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 64Mi}
                  limits: {memory: 128Mi}
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      desired=$(kubectl -n "$NS" get deploy workers -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy workers -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: workers is ${avail:-0}/${desired:-?} — the ReplicaSet can't create the rest. Why? (hint: it's not the pods)"; exit 1
      fi
      echo "PASS — every replica scheduled. Quota lives at the namespace, the error lives on the ReplicaSet — now you know where to look."
---
