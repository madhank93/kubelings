---
kind: lesson
title: The Job That Never Finishes
description: |
  A one-shot data-import Job runs forever and never reports completion, so the
  pipeline that waits on it is stuck. Diagnose why, then make the Job actually
  complete.
name: jobs
slug: jobs
createdAt: "2026-06-30"
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
      # Broken Job: the container never exits, so the Job never completes.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: batch/v1
      kind: Job
      metadata:
        name: data-import
      spec:
        backoffLimit: 4
        template:
          spec:
            restartPolicy: OnFailure
            containers:
              - name: importer
                image: ghcr.io/iximiuz/labs/busybox:latest
                # BUG: sleeps forever instead of doing the import and exiting 0.
                command: ["sh","-c","echo importing...; sleep infinity"]
      YAML
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      kubectl -n "$NS" get job data-import >/dev/null 2>&1 || {
        echo "not yet: no Job 'data-import' in $NS"; exit 1; }
      complete=$(kubectl -n "$NS" get job data-import \
        -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' 2>/dev/null)
      succeeded=$(kubectl -n "$NS" get job data-import -o jsonpath='{.status.succeeded}' 2>/dev/null)
      if [ "$complete" = "True" ] && [ "${succeeded:-0}" -ge 1 ]; then
        echo "PASS — Job data-import completed (succeeded=$succeeded)."
        exit 0
      fi
      echo "not yet: Job data-import has not completed (Complete=$complete, succeeded=${succeeded:-0})."
      echo "A Job completes only when its pod's container exits 0."
      exit 1
---
