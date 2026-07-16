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
