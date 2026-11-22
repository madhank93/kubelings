---
kind: lesson
title: 'Your first pod: what a cluster actually runs'
description: |
  Meet the cluster from the outside: nodes, namespaces, and the smallest thing
  Kubernetes runs, a pod. Start one with a single kubectl command and watch it
  go from Pending to Running.
name: first-pod
slug: first-pod
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
      kubectl -n "$NS" delete pod hello --ignore-not-found --wait=true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get pod hello >/dev/null 2>&1; then
        echo "not yet: no pod named 'hello' in namespace '$NS'"; exit 1
      fi
      image=$(kubectl -n "$NS" get pod hello -o jsonpath='{.spec.containers[0].image}')
      if [ "$image" != "nginx:1.27-alpine" ]; then
        echo "not yet: pod 'hello' runs image '$image', want nginx:1.27-alpine"; exit 1
      fi
      phase=$(kubectl -n "$NS" get pod hello -o jsonpath='{.status.phase}')
      if [ "$phase" != "Running" ]; then
        echo "not yet: pod 'hello' is $phase, not Running yet — give it a few seconds"; exit 1
      fi
      echo "PASS — your first pod is running."
---
