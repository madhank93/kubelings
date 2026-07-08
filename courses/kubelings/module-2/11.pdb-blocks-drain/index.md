---
kind: lesson
title: 'The drain that never finishes'
description: |
  Node maintenance tonight. But `kubectl drain` has been "evicting" the same two
  pods for ten minutes — a PodDisruptionBudget demands more availability than the
  deployment has replicas. Learn the PDB math and fix it so maintenance and
  availability can both be true.
name: pdb-blocks-drain
slug: pdb-blocks-drain
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
      # Make sure no node is left cordoned from a previous attempt.
      for n in $(kubectl get nodes -o name); do kubectl uncordon "${n#node/}" >/dev/null 2>&1 || true; done
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: tickets
      spec:
        replicas: 2
        selector:
          matchLabels: {app: tickets}
        template:
          metadata:
            labels: {app: tickets}
          spec:
            containers:
              - name: tickets
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      ---
      apiVersion: policy/v1
      kind: PodDisruptionBudget
      metadata:
        name: tickets-pdb
      spec:
        # BUG: minAvailable == replicas. Zero eviction headroom: drains hang forever.
        minAvailable: 2
        selector:
          matchLabels: {app: tickets}
      YAML
      kubectl -n "$NS" rollout status deploy/tickets --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get pdb tickets-pdb >/dev/null 2>&1; then
        echo "not yet: keep a PDB for tickets — fix its math, don't delete it"; exit 1
      fi
      allowed=$(kubectl -n "$NS" get pdb tickets-pdb -o jsonpath='{.status.disruptionsAllowed}' 2>/dev/null)
      if [ "${allowed:-0}" -lt 1 ]; then
        echo "not yet: PDB still allows 0 disruptions — a drain would hang (check minAvailable vs replicas)"; exit 1
      fi
      desired=$(kubectl -n "$NS" get deploy tickets -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy tickets -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: tickets not fully Available (${avail:-0}/${desired:-?})"; exit 1
      fi
      echo "PASS — disruptionsAllowed ≥ 1: maintenance can proceed AND availability holds. That's the PDB contract."
---
