---
kind: unit
title: "Incident file — killing the dashboard on Black Friday (Algolia)"
name: incident-black-friday-unit
---


> **Capstone incident file (guided study).** Told first-hand by Algolia
> engineers in a conference talk — watch it; postmortems delivered live, by
> the people who were paged, teach things text can't.
>
> Source:
> [Algolia — killing the dashboard during Black Friday (talk)](https://www.youtube.com/watch?v=Fjyg7cxRZQs)

## The shape of it

Algolia sells search — and Black Friday is to e-commerce search what New
Year's Eve is to phone networks: the one day the load curve goes vertical, and
the one day you absolutely cannot be down.

As the talk tells it, parts of their dashboard platform ran task-shaped work
as **Kubernetes Jobs** (Module 2.4) — spawn a Job per unit of work, let the
platform run it to completion. A fine pattern at normal volume. On Black
Friday, volume wasn't normal: work arrived faster than it completed, and the
Jobs machinery became the incident. The dashboard died on the day every
customer was staring at it.

This is the least exotic incident in Module 9 — no etcd surgery, no OS update,
no deprecated label. Just **arrival rate > completion rate**, sustained. Which
is exactly why it's here: this is the cascade *you're* most likely to own.

## Why Jobs-per-task melts down under peak

Walk the mechanics you already know:

**1. Every Job is control-plane load.** Each one is an API object, plus its
pod(s): admission, scheduling (Module 7's filter-score-bind, per pod),
kubelet starts, status writes, watch events to every controller. At 10× task
volume you're not running 10× compute — you're running 10× *object churn*
through the API server and etcd (compare OpenAI's aggregate-load lesson, one
door earlier in this module).

**2. Completed doesn't mean gone.** Finished Jobs and their pods **linger as
objects** until something deletes them — the same corpse problem as Evicted
pods (M8's disk-pressure drill), at scale. Without TTLs or cleanup, peak day
turns etcd and every full-list operation into an archaeology dig of dead Jobs.

**3. Retries multiply arrival rate.** A Job that fails retries per
`backoffLimit`; the callers that submitted the work time out and *resubmit*.
Under overload, failures rise, so retries rise, so load rises — the Monzo
feedback shape (lesson 9.1), with "restart storm" swapped for "retry storm."

**4. No admission control on your own front door.** Kubernetes will happily
accept Job number 50,000. Quotas (M8.3) cap *aggregate resources* per
namespace, but the queue-shaped question — "should we accept new work faster
than we finish old work?" — is one no built-in answers for you. Unbounded
acceptance is an outage with a delay on it.

## Concept checks

- Where's the knee? If tasks arrive at rate λ and complete at rate μ, what
  happens to queue length the moment λ > μ — and which Kubernetes symptoms
  would you see first? (Pending pods climbing, Job objects accumulating,
  API latency rising — in that order or all at once.)
- What single field turns "completed Jobs pile up forever" into "self-cleaning
  system"? (`ttlSecondsAfterFinished` — the TTL controller from Module 2.)
- Redesign it: keep the work, drop the per-task Job. What's the alternative?
  (A fixed pool of worker pods — a Deployment sized for peak, HPA-assisted —
  pulling from a real queue. Constant object count, elastic throughput;
  Kubernetes schedules *capacity*, the queue schedules *work*.)

## What to steal

- **Backpressure at the edge**: cap in-flight work explicitly (queue depth,
  concurrency limits, HTTP 429s) — reject early and cheaply rather than
  accept and collapse expensively.
- **Cleanup is capacity**: `ttlSecondsAfterFinished` on every Job,
  `successfulJobsHistoryLimit`/`failedJobsHistoryLimit` on every CronJob —
  from lesson one, not after the incident.
- **Load-test the platform path, not just the app path**: your Black Friday
  rehearsal must include the object churn (Jobs, pods, events), because
  that's the part that failed here.
- **Peak day is a freeze + a war room, but above all a *model***: know λ and
  μ for your queues before the day that decides your year.

*No check — study, then take the final boss.*
