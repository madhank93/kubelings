---
kind: lesson
title: 'Incident replay — all replicas on the failing node (Moonlight)'
description: |
  Replay of Moonlight's cited outage: every pod of the website landed on the
  same host — the one that then failed. 100% traffic loss with "redundant"
  replicas. Spread them so one node can never again take the whole service down.
name: incident-same-node
slug: incident-same-node
source: https://updates.moonlightwork.com/outage-post-mortem-87370
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
      # Pin all replicas to one worker — recreating Moonlight's pre-incident state.
      TARGET=$(kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
        | grep -v control-plane | head -1)
      kubectl apply -n "$NS" -f - <<YAML
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: website
      spec:
        replicas: 3
        selector:
          matchLabels: {app: website}
        template:
          metadata:
            labels: {app: website}
          spec:
            # BUG: hard pin. "It was fast on that node," someone said, months ago.
            nodeSelector:
              kubernetes.io/hostname: $TARGET
            containers:
              - name: web
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/website --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      desired=$(kubectl -n "$NS" get deploy website -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy website -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: website not fully Available (${avail:-0}/${desired:-?})"; exit 1
      fi
      nodes=$(kubectl -n "$NS" get pods -l app=website \
        -o jsonpath='{range .items[*]}{.spec.nodeName}{"\n"}{end}' 2>/dev/null | sort -u | grep -c . || true)
      if [ "${nodes:-0}" -lt 2 ]; then
        echo "not yet: all website pods are on ${nodes:-0} node(s) — one host failure still kills everything"; exit 1
      fi
      echo "PASS — replicas spread across $nodes nodes. Redundancy now means what everyone assumed it meant."
---
