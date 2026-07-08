---
kind: lesson
title: 'Everything Pending: who tainted the nodes?'
description: |
  Overnight, every new pod in the cluster stopped scheduling. Someone prepared
  the workers for a "dedicated batch pool" migration that was cancelled — but the
  taints stayed. Learn how taints repel, how tolerations permit, and how to read
  the scheduler's rejection notes.
name: taints-tolerations
slug: taints-tolerations
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
      # The cancelled migration left its mark on every worker:
      for n in $(kubectl get nodes -o name | grep -v control-plane); do
        kubectl taint node "${n#node/}" dedicated=batch:NoSchedule --overwrite
      done
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: invoices
      spec:
        replicas: 2
        selector:
          matchLabels: {app: invoices}
        template:
          metadata:
            labels: {app: invoices}
          spec:
            containers:
              - name: invoices
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # The cancelled migration's taints must be cleaned up (they'd break every
      # later lesson too — leftovers always outlive their project).
      leftover=$(kubectl get nodes -o jsonpath='{range .items[*]}{.spec.taints[?(@.key=="dedicated")].key}{"\n"}{end}' 2>/dev/null | grep -c dedicated || true)
      if [ "${leftover:-0}" -gt 0 ]; then
        echo "not yet: $leftover node(s) still carry the dedicated=batch taint — the migration was cancelled, clean it up"; exit 1
      fi
      desired=$(kubectl -n "$NS" get deploy invoices -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy invoices -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: invoices not Available (${avail:-0}/${desired:-?})"; exit 1
      fi
      echo "PASS — taints removed, pods scheduled. Taints are policy with no expiry date: audit them."
---
