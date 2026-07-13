---
kind: lesson
title: 'Drill — zombie CronJobs pile up'
description: |
  Synthetic drill of a failure pattern reported across many production
  clusters: a CronJob has been running for weeks with history limits of 50,
  and dozens of completed Jobs and their pods now clutter the namespace,
  bloat every LIST, and slow the API server's watchers. Clear the pile and
  fix the limits so it never regrows.
name: pattern-zombie-cronjobs
slug: pattern-zombie-cronjobs
createdAt: "2026-07-13"
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
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: batch/v1
      kind: CronJob
      metadata:
        name: report-gen
        labels: {app: report-gen}
      spec:
        schedule: "0 * * * *"
        # BUG: keep-everything history limits — the zombie factory.
        successfulJobsHistoryLimit: 50
        failedJobsHistoryLimit: 50
        jobTemplate:
          metadata:
            labels: {app: report-gen}
          spec:
            template:
              metadata:
                labels: {app: report-gen}
              spec:
                restartPolicy: Never
                containers:
                  - name: report
                    image: busybox:1.36
                    command: ["true"]
                    resources:
                      requests: {cpu: 10m, memory: 16Mi}
                      limits: {memory: 64Mi}
      YAML
      # "Weeks of runs": seed 25 completed Jobs directly (don't wait for
      # 25 schedule ticks). Init may wait for completion; verify may not.
      for i in $(seq -w 1 25); do
        kubectl apply -n "$NS" -f - <<JOB
      apiVersion: batch/v1
      kind: Job
      metadata:
        name: report-gen-backfill-$i
        labels: {app: report-gen}
      spec:
        template:
          metadata:
            labels: {app: report-gen}
          spec:
            restartPolicy: Never
            containers:
              - name: report
                image: busybox:1.36
                command: ["true"]
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      JOB
      done
      kubectl -n "$NS" wait --for=condition=complete job -l app=report-gen --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get cronjob report-gen >/dev/null 2>&1; then
        echo "not yet: CronJob report-gen is gone — the fix is limits + TTL, not deleting the workload"; exit 1
      fi
      ok=$(kubectl -n "$NS" get cronjob report-gen -o jsonpath='{.spec.successfulJobsHistoryLimit}')
      fail=$(kubectl -n "$NS" get cronjob report-gen -o jsonpath='{.spec.failedJobsHistoryLimit}')
      if [ "${ok:-50}" -gt 3 ] || [ "${fail:-50}" -gt 3 ]; then
        echo "not yet: history limits are $ok/$fail — cap both at 3 or less"; exit 1
      fi
      ttl=$(kubectl -n "$NS" get cronjob report-gen -o jsonpath='{.spec.jobTemplate.spec.ttlSecondsAfterFinished}')
      if [ -z "$ttl" ]; then
        echo "not yet: jobTemplate has no ttlSecondsAfterFinished — finished Jobs need an expiry, not just a cap"; exit 1
      fi
      count=$(kubectl -n "$NS" get jobs -l app=report-gen --no-headers 2>/dev/null | wc -l)
      if [ "$count" -gt 3 ]; then
        echo "not yet: $count report-gen Jobs still in the namespace — the new limits don't clean up the old pile; clear it"; exit 1
      fi
      echo "PASS — pile cleared, history capped, TTL set. The zombie factory is decommissioned."
---
