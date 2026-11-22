---
kind: lesson
title: 'Databases on Kubernetes: what a StatefulSet does not give you'
description: |
  Reading — the durability questions Kubernetes does not answer for you: async
  replication and the failover that loses committed writes, what a PVC really
  promises, why "the pod restarted somewhere else" is a data question, and when
  an operator earns its keep. Built on Gravitational's cited PostgreSQL-on-K8s
  write-up and the operator practice that grew out of it.
name: databases-on-k8s
slug: databases-on-k8s
createdAt: "2026-07-27"
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
