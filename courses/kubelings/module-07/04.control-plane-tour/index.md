---
kind: lesson
title: 'The control-plane tour: request flow, kubelet, leases'
description: |
  A guided walk through the machinery — what happens between "kubectl apply" and
  a running container, how the kubelet turns a PodSpec into containers via the
  CRI, and how leader election keeps one controller-manager in charge. Reading +
  live pokes on this cluster; no single check to pass.
name: control-plane-tour
slug: control-plane-tour
createdAt: "2026-07-07"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 120
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      kubectl -n "$NS" delete configmap quick-check --ignore-not-found
  quick_check:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get configmap quick-check >/dev/null 2>&1; then
        echo "not yet: no ConfigMap 'quick-check' in namespace '$NS'"; exit 1
      fi
      wrong=""
      for kv in q1=a q2=b q3=c; do
        k=${kv%%=*}
        got=$(kubectl -n "$NS" get configmap quick-check -o jsonpath="{.data.$k}" | tr -d ' )' | tr 'A-Z' 'a-z')
        [ "$got" = "${kv#*=}" ] || wrong="$wrong $k"
      done
      if [ -n "$wrong" ]; then
        echo "not yet: re-read the lesson and try again:$wrong"; exit 1
      fi
      echo "PASS — quick check complete."
---
