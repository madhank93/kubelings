---
kind: unit
title: "Drill — zombie CronJobs pile up"
name: pattern-zombie-cronjobs-unit
---


> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern
> write-up: [Pattern: zombie CronJobs](https://kubelings.madhan.app/incidents/pattern-zombie-cronjobs/).

## The situation

Someone complains `kubectl get pods` in this namespace "takes forever and
scrolls for a week". Look:

```sh
kubectl -n kubelings get jobs | head
kubectl -n kubelings get jobs --no-headers | wc -l      # 25 and counting
kubectl -n kubelings get pods --no-headers | wc -l      # one Completed pod each
```

`report-gen` runs hourly and has been running for weeks. Its history limits:

```sh
kubectl -n kubelings get cronjob report-gen -o jsonpath='{.spec.successfulJobsHistoryLimit}/{.spec.failedJobsHistoryLimit}'
# 50/50
```

Every finished Job object — and its `Completed` pod — stays behind, up to 100
of them. This is not just clutter: every controller, operator, and `kubectl`
LIST in the namespace pays for those objects; every watch on Jobs/Pods carries
them; etcd stores them. Multiply by fifty CronJobs across a real cluster and
the control plane is doing real work to remember garbage. (M7's watch-informers
lesson shows exactly who pays.)

## Your task

1. **Clear the existing pile** — the 25 backfill Jobs (label `app=report-gen`).
   Deleting a Job cascades to its pods.
2. **Cap the history** on the CronJob: `successfulJobsHistoryLimit` and
   `failedJobsHistoryLimit` ≤ 3.
3. **Add an expiry**: set `ttlSecondsAfterFinished` (e.g. 3600) in
   `spec.jobTemplate.spec` so every future Job self-deletes after finishing —
   even if the CronJob is ever deleted and its history-limit GC with it.

Don't delete the CronJob itself; the reports must keep shipping.

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings delete jobs -l app=report-gen
kubectl -n kubelings patch cronjob report-gen --type=merge -p '{
  "spec": {
    "successfulJobsHistoryLimit": 3,
    "failedJobsHistoryLimit": 1,
    "jobTemplate": {"spec": {"ttlSecondsAfterFinished": 3600}}
  }
}'
```

</details>

::simple-task
---
:tasks: tasks
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>


## The pattern (why this recurs everywhere)

CronJob defaults are sane (`successfulJobsHistoryLimit: 3`, `failed: 1`) — the
pile starts when someone raises them "to debug something" and never lowers
them, or templates them high org-wide. At one-per-hour, a single CronJob
manufactures ~720 objects a month. Nobody notices until LISTs slow down, an
operator OOMs re-listing pods, or etcd's object count alarms. The cleanup is
always someone's 2 a.m. `delete jobs --all` — which also nukes the run
someone else was debugging.

## Fix

```sh
# 1. the existing pile (cascades to pods):
kubectl -n kubelings delete jobs -l app=report-gen

# 2 + 3. cap history, add TTL:
kubectl -n kubelings patch cronjob report-gen --type=merge -p '{
  "spec": {
    "successfulJobsHistoryLimit": 3,
    "failedJobsHistoryLimit": 1,
    "jobTemplate": {"spec": {"ttlSecondsAfterFinished": 3600}}
  }
}'
```

## Why both mechanisms

- **History limits** are enforced by the CronJob controller — they only govern
  Jobs *owned by that CronJob*, and only while it exists.
- **`ttlSecondsAfterFinished`** is enforced by the TTL controller on the Job
  itself — it works for manually created Jobs, survives CronJob deletion, and
  cleans up failed Jobs too (after you've had time to look).

Belt and suspenders: limits keep the recent-history window, TTL guarantees
nothing lives forever.

## Prevention / takeaway

- Treat finished Jobs as logs, not objects: keep 1–3 for inspection, expire
  the rest. Your actual run history belongs in logging/metrics, not etcd.
- Ghost objects are the raw material of *other* incidents: M3's
  `pattern-pvc-terminating` drill features a forgotten pod exactly like these
  holding a volume hostage.
- Cluster hygiene check worth automating: count `Completed` pods older than a
  day; anything nonzero names a CronJob missing its TTL.

</details>
