---
kind: lesson
title: 'Manifests: write down what you want, then apply it'
description: |
  Real clusters are not built with one-off commands. Write a Pod as a YAML
  manifest (apiVersion, kind, metadata, spec), give it labels, and create it
  with kubectl apply, the command every later lesson uses.
name: pod-manifests
slug: pod-manifests
createdAt: "2026-11-22"
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
      kubectl -n "$NS" delete pod web --ignore-not-found --wait=true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get pod web >/dev/null 2>&1; then
        echo "not yet: no pod named 'web' in namespace '$NS'"; exit 1
      fi
      if [ -z "$(kubectl -n "$NS" get pod web -o jsonpath='{.metadata.annotations.kubectl\.kubernetes\.io/last-applied-configuration}')" ]; then
        echo "not yet: pod 'web' was not created with 'kubectl apply -f' — delete it and apply your manifest"; exit 1
      fi
      app=$(kubectl -n "$NS" get pod web -o jsonpath='{.metadata.labels.app}')
      tier=$(kubectl -n "$NS" get pod web -o jsonpath='{.metadata.labels.tier}')
      if [ "$app" != "web" ] || [ "$tier" != "frontend" ]; then
        echo "not yet: pod 'web' needs labels app=web and tier=frontend (has app='$app' tier='$tier')"; exit 1
      fi
      port=$(kubectl -n "$NS" get pod web -o jsonpath='{.spec.containers[0].ports[0].containerPort}')
      if [ "$port" != "80" ]; then
        echo "not yet: the container should declare containerPort 80 (has '$port')"; exit 1
      fi
      mem=$(kubectl -n "$NS" get pod web -o jsonpath='{.spec.containers[0].resources.limits.memory}')
      if [ "$mem" != "128Mi" ]; then
        echo "not yet: the container should have a memory limit of 128Mi (has '$mem')"; exit 1
      fi
      if [ "$(kubectl -n "$NS" get pod web -o jsonpath='{.status.phase}')" != "Running" ]; then
        echo "not yet: pod 'web' is not Running yet"; exit 1
      fi
      echo "PASS — declared in YAML, labelled, and applied."
---
