---
kind: lesson
title: 'What the scheduler actually does (bypass it to find out)'
description: |
  The scheduler feels like a black box. It isn't — its entire job is to write one
  field: spec.nodeName. Prove it by scheduling a pod yourself, bypassing the
  scheduler entirely, then reason about filter-and-score placement.
name: scheduler-nodename
slug: scheduler-nodename
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
      # A pod with NO scheduler assignment: schedulerName points at one that
      # doesn't exist, so the default scheduler ignores it -> stays Pending until
      # a human writes nodeName. This makes the scheduler's real job undeniable.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: Pod
      metadata:
        name: manual-sched
        labels: {app: manual-sched}
      spec:
        schedulerName: no-such-scheduler
        containers:
          - name: web
            image: nginx:1.27-alpine
            resources:
              requests: {cpu: 10m, memory: 32Mi}
              limits: {memory: 128Mi}
      YAML
      sleep 3 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      node=$(kubectl -n "$NS" get pod manual-sched -o jsonpath='{.spec.nodeName}' 2>/dev/null)
      if [ -z "$node" ]; then
        echo "not yet: manual-sched has no spec.nodeName — no scheduler will place it. Assign a node yourself."; exit 1
      fi
      # It must be a real worker (not control-plane, which is tainted).
      if kubectl get node "$node" >/dev/null 2>&1; then :; else
        echo "not yet: spec.nodeName='$node' is not a real node in this cluster"; exit 1
      fi
      phase=$(kubectl -n "$NS" get pod manual-sched -o jsonpath='{.status.phase}' 2>/dev/null)
      if [ "$phase" != "Running" ]; then
        echo "not yet: manual-sched is $phase on node '$node' — pick a schedulable worker so the kubelet can run it"; exit 1
      fi
      echo "PASS — you did the scheduler's whole job by writing one field. Everything else it does is deciding WHICH node to write."
---
