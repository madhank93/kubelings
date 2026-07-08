---
title: "Algolia: killing the dashboard on Black Friday"
description: "[REAL] — Jobs-shaped work met peak-day traffic; arrival rate beat completion rate and the accumulated churn took the dashboard down at peak. Kubelings capstone reading."
---

> **[REAL] incident** — told first-hand by Algolia engineers:
> [conference talk](https://www.youtube.com/watch?v=Fjyg7cxRZQs).
> **Kubelings capstone reading:** lesson `incident-black-friday` (Module 9 — War Stories).

## Situation

Algolia sells search; Black Friday is e-commerce's vertical-load day — the one
day you cannot be down. Parts of their dashboard platform ran task-shaped work
as **Kubernetes Jobs**: one Job per unit of work. Fine at normal volume. On
Black Friday, work arrived faster than it completed — and the Jobs machinery
itself became the incident. The dashboard died on the day every customer was
watching it.

## Blast radius

- Dashboard down during peak — maximal customer visibility.
- The platform (API objects, scheduling, etcd) was the bottleneck, not compute.

## Root cause chain

1. **Every Job is control-plane load** — API object + pod(s): admission,
   scheduling, status writes, watch events. 10× task volume = 10× object
   churn, not just 10× compute.
2. **Completed ≠ gone** — finished Jobs and pods linger as objects unless
   TTLs/cleanup delete them; peak day turned list operations into archaeology.
3. **Retries multiply arrivals** — failures rise under overload, `backoffLimit`
   retries plus caller resubmits rise with them: a retry storm.
4. **No admission control on the front door** — nothing asked "should we accept
   new work faster than we finish old work?" Unbounded acceptance is an outage
   with a delay on it.

## Fix & prevention

- **Backpressure at the edge**: explicit in-flight caps, queue depth limits,
  cheap early rejection (429) over expensive collapse.
- **Cleanup as capacity**: `ttlSecondsAfterFinished` on every Job,
  history limits on every CronJob — from day one.
- **A worker-pool redesign** for high-volume task work: fixed Deployment + HPA
  pulling from a real queue — Kubernetes schedules *capacity*, the queue
  schedules *work*.
- Load-test the **platform path** (object churn), not just the app path.

## What it teaches

| Concept | Kubelings module |
|---|---|
| the full story (guided study) | M9 — `incident-black-friday` (reading) |
| Jobs & CronJob hygiene | M2 — `jobs`, `cronjobs` |
| aggregate control-plane load | M9 — `incident-openai-cascade` |
| object corpses | M8 — `pattern-disk-pressure` |
