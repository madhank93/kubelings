---
title: "Pattern: zombie CronJobs"
description: "[PATTERN] Synthetic composite — keep-everything history limits let finished Jobs and Completed pods pile up until LISTs, watches, and etcd feel it."
---

> **[PATTERN] scenario** — a synthetic composite of a failure mode reported
> across many production clusters. **No specific company**; details are
> representative, not cited. (Real, cited incidents are marked `[REAL]` in the
> [Incident Library](/catalog/).)

## Situation

`kubectl get pods` in a namespace takes seconds and scrolls through hundreds
of `Completed` pods. An operator restarts and OOMs re-listing Jobs. etcd
object counts creep up quarter over quarter. Somewhere, an hourly CronJob has
been running for months with `successfulJobsHistoryLimit: 50` — every run
leaves a Job object and its pod behind, and nothing ever deletes them.

## Root cause

CronJob **history limits raised and forgotten**. The defaults are sane (3
successful / 1 failed); someone raised them "temporarily, to debug" or a
platform template shipped them high. At one run per hour, one CronJob
manufactures ~720 Jobs a month. Every finished Job and its `Completed` pod is
a live API object: it costs etcd storage, inflates every LIST in the
namespace, and rides along in every watch cache and informer re-list —
control-plane load spent remembering garbage.

The pile also breeds secondary incidents: forgotten pods holding PVCs hostage
(see [Pattern: PVC stuck Terminating](/incidents/pattern-pvc-terminating/)),
and 2 a.m. `delete jobs --all` cleanups that nuke the one Job someone was
debugging.

## Diagnosis

```sh
# who's hoarding:
kubectl get cronjobs -A -o custom-columns='NS:.metadata.namespace,NAME:.metadata.name,OK:.spec.successfulJobsHistoryLimit,FAIL:.spec.failedJobsHistoryLimit'
# the pile:
kubectl get jobs -A --no-headers | wc -l
kubectl get pods -A --field-selector=status.phase=Succeeded --no-headers | wc -l
```

## Fix

```sh
# clear the existing pile (cascades to pods):
kubectl delete jobs -l app=<name>

# cap history AND add an expiry to the template:
kubectl patch cronjob <name> --type=merge -p '{
  "spec": {
    "successfulJobsHistoryLimit": 3,
    "failedJobsHistoryLimit": 1,
    "jobTemplate": {"spec": {"ttlSecondsAfterFinished": 3600}}
  }
}'
```

Both mechanisms, deliberately: history limits are enforced by the CronJob
controller and only govern Jobs it owns; `ttlSecondsAfterFinished` is enforced
by the TTL controller on each Job — it covers manual Jobs and survives the
CronJob's deletion.

## Prevention

- Finished Jobs are logs, not objects: keep 1–3 for inspection, expire the
  rest. Run history belongs in your logging stack, not etcd.
- Policy-check `ttlSecondsAfterFinished` on every Job/CronJob template
  (admission policy — Kubelings M6 covers enforcing exactly this).
- Hygiene alert: any `Completed` pod older than a day names a CronJob missing
  its TTL.

## What it teaches

| Concept | Kubelings module |
|---|---|
| Job/CronJob lifecycle, history limits, TTL controller | M2 Workloads (`pattern-zombie-cronjobs`) |
| Watch/LIST cost of object bloat | M7 Internals (`watch-informers`) |
