---
kind: unit
title: "Drill — the noisy neighbor"
name: pattern-noisy-neighbor-unit
---


> **Drill** — synthetic composite of a pattern behind several cited incidents in
> the [Incident Library](https://kubelings.madhan.app/reference/incident-library/)
> (Omio's throttling deep-dive, Buffer's limits experiment, Datadog's talk all
> orbit it). No single company; every company.

## The situation

The ticket reads: *"quotes-api p99 went from 40ms to 2.3s. No deploys. No
errors. It's not down, it's just… wading through glue."*

Nothing crashed — so Module 1's triage loop finds nothing. This is the other
family of production pain: **contention**. Two tenants on this node:

- `media-encoder` — someone's transcoder, busy-looping a full core, declaring
  **zero** resources. To the scheduler it weighs nothing; it'll happily place
  more work next to it. (You met this sin in the QoS lesson — here's the damage
  it does to *others*.)
- `quotes-api` — declares `limits.cpu: 20m`. One-fiftieth of a core. The CFS
  scheduler enforces that in 100ms windows: use your 2ms budget, **sleep the
  remaining 98ms**. Even with the node idle, this pod would crawl — with the
  encoder feasting next door it's throttling *and* starving.

Two resource lies, one slow service, no red statuses anywhere.

## Your task

Make both tenants honest:

1. `media-encoder`: real CPU requests **and** a limit — it pays for what it
   uses, and the scheduler finally sees its weight.
2. `quotes-api`: a CPU limit that permits actual work (≥ `100m`; or reason
   about dropping the CPU limit entirely — see the solution's debate).
3. Both Available.

```sh
kubectl -n kubelings top pods 2>/dev/null || echo "(metrics-server still warming)"
kubectl -n kubelings get pods -o custom-columns=NAME:.metadata.name,QOS:.status.qosClass
```

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings set resources deploy/media-encoder \
  --requests=cpu=500m,memory=64Mi --limits=cpu=500m,memory=128Mi
kubectl -n kubelings set resources deploy/quotes-api \
  --requests=cpu=100m,memory=32Mi --limits=cpu=500m,memory=128Mi
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


## The mechanics, precisely

**CPU requests** are proportional *shares* — under contention, the kernel divides
CPU by the ratio of requests. Declaring none = owning ~nothing in the fight you
started.

**CPU limits** are CFS *quota*: `limit × 100ms` of runtime per 100ms period,
then a hard throttle to zero until the next period. Throttling is per-window
stop-and-go — which is why it manifests as *latency spikes*, not lower
throughput. `20m` = 2ms of allowed run per 100ms. That's the 2.3s p99 right
there.

## Fix

The hint's two commands. Note what each does:

- Encoder becomes **Guaranteed-ish** and visible: scheduler stops overpacking
  its node, kernel caps its feast at 500m.
- Victim keeps a modest request (its fair share floor) and a limit with real
  headroom.

## The "should CPU limits exist at all?" debate — the honest version

Buffer removed CPU limits fleet-wide and got faster services (cited in the
library). The logic: requests already guarantee fairness under contention;
limits *only* add throttling above your slice, even when the node is idle.
Counter-logic: limits make worst-case latency predictable, contain runaway
loops, and mixed-criticality nodes want ceilings on the untrusted.

Defensible policy most fleets land on: **requests always, honest; CPU limits
only where you can articulate why** (untrusted/batch tiers) — **memory limits
always** (memory isn't compressible; the OOM killer is the alternative).

## Prevention

- LimitRange per namespace injecting default requests — makes "declared nothing"
  impossible (Module 6 energy).
- Alert on `container_cpu_cfs_throttled_periods_total` ratio > ~25% — throttling
  is measurable *before* users feel it.
- "Slow but not down, no deploy" → check throttling and neighbors *first*,
  dashboards second.

</details>
