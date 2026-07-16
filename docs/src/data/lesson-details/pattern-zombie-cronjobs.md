> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern

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
