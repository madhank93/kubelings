---
kind: unit
title: "CronJob Pileup: fix the concurrencyPolicy"
name: cronjobs-unit
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

::simple-task
---
:tasks: tasks
:name: verify_done
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>


## Root cause

`report` runs every minute but each run lasts ~90s, and `concurrencyPolicy: Allow`
lets a new run start on schedule regardless of in-flight runs. Overlapping Jobs
accumulate until resources are exhausted.

## Fix

```sh
kubectl -n kubelings patch cronjob report --type=merge \
  -p '{"spec":{"concurrencyPolicy":"Forbid"}}'
```

- `Forbid` — skip the new run if the previous one is still active.
- `Replace` — cancel the running one and start the new one instead.

## Verify

```sh
kubectl -n kubelings get cronjob report \
  -o jsonpath='{.spec.concurrencyPolicy}{"\n"}'
kubectl -n kubelings get jobs   # runs no longer stack up
```

</details>
