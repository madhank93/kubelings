---
title: "Universe: restartPolicy Never did not mean never"
description: "[REAL] 2017 — a reindex Job written as run-once retried forever; the greedy retries crashed nodes and took the whole cluster's search and reporting with them. Runnable Kubelings replay."
---

> **[REAL] incident** — cited from Universe's public status-page postmortem:
> [Search and Reporting Outage](http://status.universe.com/incidents/115n3vxqwzcf).
> **Runnable replay:** Kubelings lesson `incident-job-restartpolicy`
> (Module 2 — Workloads).

## Situation

Universe (ticketing platform) shipped a **fix**: a reindex Job to repair broken
search indexing. Within minutes, search and reporting were degraded — and so was
the cluster UI, because the cluster itself was unhealthy. Resolved 18 minutes
later by deleting the Job.

## Blast radius

- Search and reporting APIs degraded cluster-wide.
- Nodes pushed into unhealthy states; the failure spread beyond the workload that
  caused it.
- Recovery required a human to identify and delete the offending Job; nodes
  auto-healed once it was gone.

## Root cause chain

1. **The Job was written as "run once".** Its author set
   `restartPolicy: Never`, believing that bounded the work.
2. **That field belongs to the pod, not the Job.** `Never` tells the kubelet not
   to restart the *container in place*. The Job controller responds to a failed
   pod by creating **another pod** — which is a different retry loop entirely,
   governed by `backoffLimit`.
3. **The retry loop was unbounded and greedy.** Each attempt was a fresh pod
   consuming node resources; nothing capped attempts (`backoffLimit`), wall-clock
   (`activeDeadlineSeconds`) or the container's appetite (`resources.limits`).
4. **Node-level damage cascaded.** Resource exhaustion on one node moved
   workloads elsewhere, spreading the pressure — a workload bug became a cluster
   incident.

The postmortem's own words: `RestartPolicy=Never` "applies to the pod, not to
the job."

## Fix & prevention

- **Bound every Job twice:** `backoffLimit` (attempts) *and*
  `activeDeadlineSeconds` (wall-clock). One catches crashes, the other catches
  hangs.
- **Limits on batch containers**, always — a retry storm without a ceiling is a
  self-inflicted resource-exhaustion attack.
- **`ttlSecondsAfterFinished`** so failed/completed Jobs and their pods collect
  themselves instead of piling up.
- **Alert on `Job.status.failed`**, not only on pod dashboards — a Job burning
  retries at 3am is invisible in a per-pod view.
- Put both bounds in the Job template every team copies from; the scaffold's
  defaults are what ends up in production.

## What it teaches

| Concept | Kubelings module |
|---|---|
| Job retry semantics vs pod restartPolicy | M2 — `incident-job-restartpolicy` (runnable) |
| Job completion and failure conditions | M2 — `jobs` |
| Job/pod pile-up and TTL | M2 — `pattern-zombie-cronjobs` |
| Resource limits and node blast radius | M2 — `qos-classes`, M8 — `incident-node-oom` |
