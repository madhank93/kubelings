---
kind: challenge

title: "CronJob Pileup: fix the concurrencyPolicy"
description: |
  A every-minute CronJob runs longer than a minute, so overlapping runs stack up
  and exhaust resources. Configure the CronJob so a new run never starts while the
  previous one is still going.

categories:
- kubernetes

tagz:
- cka
- workloads
- cronjob

difficulty: easy

createdAt: 2026-06-30

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
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: batch/v1
      kind: CronJob
      metadata:
        name: report
      spec:
        schedule: "* * * * *"        # every minute
        concurrencyPolicy: Allow     # BUG: overlapping runs pile up
        jobTemplate:
          spec:
            template:
              spec:
                restartPolicy: Never
                containers:
                  - name: report
                    image: ghcr.io/iximiuz/labs/busybox:latest
                    # Runs ~90s — longer than the 60s schedule interval.
                    command: ["sh","-c","echo working; sleep 90"]
      YAML

  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      kubectl -n "$NS" get cronjob report >/dev/null 2>&1 || {
        echo "not yet: no CronJob 'report' in $NS"; exit 1; }
      pol=$(kubectl -n "$NS" get cronjob report -o jsonpath='{.spec.concurrencyPolicy}')
      if [ "$pol" = "Forbid" ] || [ "$pol" = "Replace" ]; then
        echo "PASS — concurrencyPolicy is '$pol'; overlapping runs can no longer pile up."
        exit 0
      fi
      echo "not yet: concurrencyPolicy is '$pol' — overlapping runs still stack up."
      echo "Use 'Forbid' (skip new run while one is active) or 'Replace'."
      exit 1
---

## The situation

The `report` CronJob in `kubelings` is scheduled **every minute**, but each run
takes ~90 seconds. With `concurrencyPolicy: Allow`, Kubernetes starts a fresh run
on schedule even though the previous one is still running — so active Jobs pile up
and pods accumulate until the namespace runs out of resources.

## Your task

Configure `report` so a new run **does not start while the previous run is still
active** (set `concurrencyPolicy` to `Forbid`, or `Replace` if you'd rather kill
the old run and start fresh).

```sh
kubectl -n kubelings get cronjob report -o yaml | grep concurrencyPolicy
kubectl -n kubelings get jobs
```

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch cronjob report --type=merge \
  -p '{"spec":{"concurrencyPolicy":"Forbid"}}'
```

</details>
