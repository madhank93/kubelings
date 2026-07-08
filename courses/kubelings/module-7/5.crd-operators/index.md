---
kind: lesson
title: 'CRDs: teach the API server a new noun'
description: |
  A team shipped manifests for a BackupSchedule resource — and the apply fails
  with "no matches for kind". Nothing is broken: the cluster just doesn't know
  that noun yet. Install the CustomResourceDefinition, watch the API grow a new
  endpoint, and see how CRD + controller = operator.
name: crd-operators
slug: crd-operators
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
      kubectl delete crd backupschedules.kubelings.dev --ignore-not-found
      # The scenario is an absence: the team's CR manifest has nowhere to land.
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl get crd backupschedules.kubelings.dev >/dev/null 2>&1; then
        echo "not yet: no CRD backupschedules.kubelings.dev — the cluster still doesn't know the noun"; exit 1
      fi
      est=$(kubectl get crd backupschedules.kubelings.dev -o jsonpath='{.status.conditions[?(@.type=="Established")].status}' 2>/dev/null)
      if [ "$est" != "True" ]; then
        echo "not yet: CRD exists but isn't Established — check .status.conditions for what the API server rejected"; exit 1
      fi
      sched=$(kubectl -n "$NS" get backupschedule nightly-etcd -o jsonpath='{.spec.schedule}' 2>/dev/null)
      if [ -z "$sched" ]; then
        echo "not yet: no BackupSchedule named nightly-etcd in kubelings — apply the team's CR"; exit 1
      fi
      if [ "$sched" != "0 2 * * *" ]; then
        echo "not yet: nightly-etcd has schedule '$sched', expected '0 2 * * *'"; exit 1
      fi
      echo "PASS — the API server now serves /apis/kubelings.dev/v1/backupschedules like it was born with it. CRD in; a controller would make it an operator."
---
