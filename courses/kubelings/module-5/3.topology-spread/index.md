---
kind: lesson
title: 'Topology spread: balance, not just separation'
description: |
  Six replicas of the session store — five on one node, one on another.
  Anti-affinity can't fix this: it only says "not together", not "evenly".
  Learn topologySpreadConstraints, maxSkew math, and when soft beats hard.
name: topology-spread
slug: topology-spread
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
      # Stack 5 of 6 replicas onto one worker (nodeSelector), then release the
      # pin so the learner starts from an unbalanced-but-unpinned state.
      TARGET=$(kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
        | grep -v control-plane | head -1)
      kubectl apply -n "$NS" -f - <<YAML
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: sessions
      spec:
        replicas: 5
        selector:
          matchLabels: {app: sessions}
        template:
          metadata:
            labels: {app: sessions}
          spec:
            nodeSelector:
              kubernetes.io/hostname: $TARGET
            containers:
              - name: sessions
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/sessions --timeout=180s
      # Release the pin WITHOUT restarting pods: existing 5 stay stacked, and a
      # 6th replica appears wherever the scheduler likes. Balance is now on you.
      kubectl -n "$NS" patch deploy sessions --type=json -p '[
        {"op":"remove","path":"/spec/template/spec/nodeSelector"},
        {"op":"replace","path":"/spec/replicas","value":6}
      ]' || true
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      desired=$(kubectl -n "$NS" get deploy sessions -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy sessions -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: sessions not fully Available (${avail:-0}/${desired:-?})"; exit 1
      fi
      # Balance check: per-node counts, max-min must be <= 2 across used nodes,
      # and at least 2 nodes must carry pods.
      counts=$(kubectl -n "$NS" get pods -l app=sessions \
        -o jsonpath='{range .items[*]}{.spec.nodeName}{"\n"}{end}' 2>/dev/null | sort | uniq -c | awk '{print $1}')
      nnodes=$(grep -c . <<<"$counts" || true)
      if [ "${nnodes:-0}" -lt 2 ]; then
        echo "not yet: all sessions pods on one node"; exit 1
      fi
      mx=$(sort -n <<<"$counts" | tail -1); mn=$(sort -n <<<"$counts" | head -1)
      skew=$((mx - mn))
      if [ "$skew" -gt 2 ]; then
        echo "not yet: node balance is $counts (skew $skew) — declare a spread constraint and re-roll"; exit 1
      fi
      if [ -z "$(kubectl -n "$NS" get deploy sessions \
          -o jsonpath='{.spec.template.spec.topologySpreadConstraints}' 2>/dev/null)" ]; then
        echo "not yet: balance without a declared constraint is luck — add topologySpreadConstraints"; exit 1
      fi
      echo "PASS — spread declared and skew ≤ 2. Balance survives the next reschedule because it's policy, not luck."
---
