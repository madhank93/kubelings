---
kind: lesson
title: 'Incident file — the Pi-Day outage (Reddit, 2023)'
description: |
  Guided study of Reddit's cited Pi-Day outage: a routine Kubernetes upgrade on
  their oldest cluster renamed one node label — and the CNI's route reflectors,
  selecting nodes by the old name, went empty. Pod networking collapsed
  cluster-wide. A war story about deprecations, snowflake clusters, and why
  restores get tested before they're needed.
name: incident-reddit-piday
slug: incident-reddit-piday
source: https://www.reddit.com/r/RedditEng/comments/11xx5o0/you_broke_reddit_the_piday_outage/
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
