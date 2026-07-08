---
kind: lesson
title: 'etcd: the one source of truth'
description: |
  Every object you have ever created lives as a key in etcd. Write a ConfigMap,
  then find it byte-for-byte inside etcd at /registry/... — and understand why
  "the API server is stateless" and "back up etcd or lose everything" are the
  same fact.
name: etcd-truth
slug: etcd-truth
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
      # Clean slate for the sentinel the learner will create.
      kubectl -n "$NS" delete configmap etcd-proof --ignore-not-found
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      val=$(kubectl -n "$NS" get configmap etcd-proof -o jsonpath='{.data.marker}' 2>/dev/null)
      if [ "$val" != "i-was-here" ]; then
        echo "not yet: create ConfigMap 'etcd-proof' in $NS with data key marker=i-was-here, then find it in etcd"; exit 1
      fi
      echo "PASS — the object exists via the API AND (as the unit shows) sits at /registry/configmaps/$NS/etcd-proof in etcd. Same fact, two views."
---
