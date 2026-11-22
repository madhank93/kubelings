---
kind: lesson
title: 'Incident file — the OS under the cluster (Datadog, 2023)'
description: |
  Guided study of Datadog's cited 2023 outage: a routine systemd security update
  auto-applied across the fleet restarted systemd-networkd, which silently
  deleted the network routes Cilium had installed — and tens of thousands of
  nodes across five regions dropped off the network in the same hour. The layer
  below Kubernetes can take down everything above it.
name: incident-datadog-cilium
slug: incident-datadog-cilium
source: https://www.datadoghq.com/blog/2023-03-08-multiregion-infrastructure-connectivity-issue/
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
      for kv in q1=b q2=a q3=c; do
        k=${kv%%=*}
        got=$(kubectl -n "$NS" get configmap quick-check -o jsonpath="{.data.$k}" | tr -d ' )' | tr 'A-Z' 'a-z')
        [ "$got" = "${kv#*=}" ] || wrong="$wrong $k"
      done
      if [ -n "$wrong" ]; then
        echo "not yet: re-read the lesson and try again:$wrong"; exit 1
      fi
      echo "PASS — quick check complete."
---
