---
kind: lesson
title: 'cluster-admin for a bot: scope it down'
description: |
  The CI bot needs to read pods in one namespace. Someone gave it cluster-admin
  — write access to everything, everywhere, forever. Replace the god-grant with
  a least-privilege Role and prove the difference with kubectl auth can-i.
name: rbac-least-privilege
slug: rbac-least-privilege
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
      kubectl -n "$NS" create serviceaccount ci-bot --dry-run=client -o yaml | kubectl apply -f -
      # The Friday-afternoon grant: "just make it work"
      kubectl create clusterrolebinding ci-bot-admin \
        --clusterrole=cluster-admin \
        --serviceaccount="$NS:ci-bot" \
        --dry-run=client -o yaml | kubectl apply -f -
      # Something for the bot to legitimately look at:
      kubectl -n "$NS" create deployment build-info --image=nginx:1.27-alpine \
        --dry-run=client -o yaml | kubectl apply -f -
      kubectl -n "$NS" rollout status deploy/build-info --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      SA="system:serviceaccount:$NS:ci-bot"
      # The god-grant must be gone.
      if kubectl get clusterrolebinding ci-bot-admin >/dev/null 2>&1; then
        echo "not yet: clusterrolebinding ci-bot-admin still exists — the bot is still cluster-admin"; exit 1
      fi
      # Must NOT be able to do dangerous things:
      if [ "$(kubectl auth can-i delete deployments --as="$SA" -n kube-system 2>/dev/null)" = "yes" ]; then
        echo "not yet: ci-bot can still delete deployments in kube-system"; exit 1
      fi
      if [ "$(kubectl auth can-i get secrets --as="$SA" -n "$NS" 2>/dev/null)" = "yes" ]; then
        echo "not yet: ci-bot can read Secrets — a read-only pod-watcher doesn't need that"; exit 1
      fi
      # Must still be able to do its actual job:
      if [ "$(kubectl auth can-i get pods --as="$SA" -n "$NS" 2>/dev/null)" != "yes" ]; then
        echo "not yet: ci-bot can no longer read pods in $NS — you scoped past its real job"; exit 1
      fi
      if [ "$(kubectl auth can-i list pods --as="$SA" -n "$NS" 2>/dev/null)" != "yes" ]; then
        echo "not yet: ci-bot needs list on pods in $NS too"; exit 1
      fi
      echo "PASS — the bot can do its job and nothing else. That asymmetry is the entire point of RBAC."
---
