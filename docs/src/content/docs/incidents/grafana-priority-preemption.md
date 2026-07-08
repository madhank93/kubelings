---
title: "Grafana Labs: outage by Pod Priority"
description: "[REAL] Jul 2019 — introducing PriorityClasses with the wrong defaults let Kubernetes preempt production pods to make room. Runnable Kubelings replay."
---

> **[REAL] incident** — cited from Grafana Labs' public write-up:
> [How a production outage was caused using Kubernetes pod priorities](https://grafana.com/blog/2019/07/24/how-a-production-outage-was-caused-using-kubernetes-pod-priorities/).
> **Runnable replay:** Kubelings lesson `incident-priority-preemption` (Module 5 — Scheduling & Placement).

## Situation

Grafana Labs introduced PriorityClasses to protect important workloads — a
best-practice hardening change. Rolling it out took down production ingestion
in their hosted metrics platform.

## Blast radius

- Production Cortex (metrics ingestion) pods **evicted by the scheduler itself**
  to make room for other pods.
- Ingestion disrupted — the outage was caused by the mechanism meant to prevent it.

## Root cause chain

1. **Trigger** — new PriorityClasses applied to *some* workloads first; rollout
   was gradual.
2. **The default nobody set** — pods with no PriorityClass have priority `0`.
   The moment any pod has a positive priority, everything unclassified becomes
   **lower-priority prey**.
3. **Preemption** — on a full cluster, the scheduler makes room for a
   higher-priority pending pod by **evicting** lower-priority running pods.
   Production pods that hadn't been migrated yet were outranked — and evicted.
4. **Capacity pressure** — the cluster was tight, so preemption wasn't a
   theoretical path; it fired immediately.

Priority is relative: assigning it to one workload silently *demotes* every
workload you haven't classified yet.

## Fix & prevention (what the write-up teaches)

- Roll out priorities **cluster-complete or not at all**: define the full tier
  ladder (critical / default / batch) and a `globalDefault` class before any
  workload opts in.
- Use `preemptionPolicy: Never` for classes that should queue-jump the
  scheduler but never evict others.
- Keep headroom: preemption is a symptom of a full cluster; capacity planning
  is the quieter fix (see `quota-exhausted`, M8).

## What it teaches

| Concept | Kubelings module |
|---|---|
| PriorityClass & preemption mechanics | M5 — `incident-priority-preemption` (runnable) |
| who dies first under pressure | M2 — `qos-classes` |
| aggregate capacity limits | M8 — `quota-exhausted` |
