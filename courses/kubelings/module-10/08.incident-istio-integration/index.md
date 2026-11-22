---
kind: lesson
title: 'Incident file — how we failed to integrate Istio (Exponea)'
description: |
  Read-through of Exponea's cited write-up: a service-mesh rollout abandoned
  after the sidecar broke Jobs, StatefulSets, gRPC load balancing and graceful
  shutdown one after another. No tasks; this is the platform-engineering
  judgement call — what a mesh actually costs, which of your workloads it
  changes, and how to decide whether to adopt one at all.
name: incident-istio-integration
slug: incident-istio-integration
source: https://medium.com/@jakubkulich/sailing-with-the-istio-through-the-shallow-water-8ae81668381e
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
