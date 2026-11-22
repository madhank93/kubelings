---
kind: lesson
title: 'Incident file — killing the dashboard on Black Friday (Algolia)'
description: |
  Guided study of Algolia's cited war story: a Jobs-based workload meets the
  biggest traffic day of the year, work piles up faster than it drains, and the
  accumulated load takes the platform down at peak. The overload playbook —
  backpressure, cleanup, and capacity math — as a story with a date on it.
name: incident-black-friday
slug: incident-black-friday
source: https://www.youtube.com/watch?v=Fjyg7cxRZQs
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
      for kv in q1=a q2=c q3=b; do
        k=${kv%%=*}
        got=$(kubectl -n "$NS" get configmap quick-check -o jsonpath="{.data.$k}" | tr -d ' )' | tr 'A-Z' 'a-z')
        [ "$got" = "${kv#*=}" ] || wrong="$wrong $k"
      done
      if [ -n "$wrong" ]; then
        echo "not yet: re-read the lesson and try again:$wrong"; exit 1
      fi
      echo "PASS — quick check complete."
---
