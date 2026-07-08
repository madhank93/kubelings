---
kind: lesson
title: 'Incident replay — the node that OOMed itself (Blue Matador)'
description: |
  Replay of Blue Matador's cited postmortem: pods with no memory limits grew
  until the NODE ran out, and the kernel OOM-killer started shooting processes —
  including system daemons — at random. Learn node-level OOM vs pod OOM, and cap
  the workload before the kernel does it for you.
name: incident-node-oom
slug: incident-node-oom
source: https://www.bluematador.com/blog/post-mortem-kubernetes-node-oom
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
      # A memory-hungry workload with requests but NO limit -> Burstable, free to
      # grow until the node itself is starved (node-level OOM, not pod OOM).
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: log-shipper
      spec:
        replicas: 2
        selector:
          matchLabels: {app: log-shipper}
        template:
          metadata:
            labels: {app: log-shipper}
          spec:
            containers:
              - name: shipper
                image: nginx:1.27-alpine
                # BUG: request set, NO limit. Nothing caps growth at the pod level.
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/log-shipper --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # Every container must now declare a memory LIMIT (the node's protection).
      missing=$(kubectl -n "$NS" get pods -l app=log-shipper \
        -o jsonpath='{range .items[*]}{.spec.containers[0].resources.limits.memory}{"\n"}{end}' 2>/dev/null \
        | grep -c '^$' || true)
      any=$(kubectl -n "$NS" get pods -l app=log-shipper -o name 2>/dev/null | grep -c . || true)
      if [ "${any:-0}" -eq 0 ] || [ "${missing:-1}" -gt 0 ]; then
        echo "not yet: some log-shipper pods still have no memory limit — the node is still unprotected"; exit 1
      fi
      avail=$(kubectl -n "$NS" get deploy log-shipper -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ "${avail:-0}" -lt 2 ]; then
        echo "not yet: log-shipper not fully Available"; exit 1
      fi
      echo "PASS — every container capped. Now a runaway kills only itself (pod OOM), never the node."
---
