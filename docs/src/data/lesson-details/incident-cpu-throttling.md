## The situation

The `pricing` service's p95 latency is terrible. Every dashboard says it's
fine: pod Running, zero restarts, no OOMKills, and CPU usage a fraction of a
core. The OOMKill lesson (2.7) taught you memory limits kill loudly. CPU
limits are the opposite personality: **they never kill anything — they make
everything quietly late.**

Ask the kernel directly. Every container carries its own cgroup accounting,
readable from inside:

```sh
kubectl -n kubelings exec deploy/pricing -- cat /sys/fs/cgroup/cpu.stat
```

```
usage_usec 2154321
nr_periods 412
nr_throttled 371
throttled_usec 29873211
```

Read `nr_throttled` against `nr_periods`: in ~90% of scheduling windows, this
process wanted CPU and the kernel said **wait**. (`cpu.stat` is cgroup v2 —
current kind/EKS/GKE nodes; on old v1 nodes the same numbers live in
`/sys/fs/cgroup/cpu/cpu.stat`.) Run it twice a few seconds apart —
`nr_throttled` climbing is the smoking gun.

## Why "low average CPU" and "constantly throttled" are both true

A CPU limit is enforced by the kernel's **CFS bandwidth controller** in fixed
**100ms windows**. `limits.cpu: 50m` means: *in every 100ms window, this
cgroup may run for 5ms; then it's frozen until the next window.*

`pricing` burns ~200ms of CPU per request burst, then idles. Do the math the
dashboard doesn't:

- The burst needs 200ms of compute → at 5ms per 100ms window, it takes
  **~40 windows ≈ 4 seconds** of wall clock, frozen 95% of the time.
- Then it sleeps. Average usage: a few percent of a core. Utilization graphs
  shrug; every user waits four seconds.

That's the trap Omio documented (and Buffer, and half the industry): **CPU
averages hide burst starvation.** Multi-threaded apps hit it even harder — a
400m limit across 4 busy threads burns the whole window's quota in 10ms and
freezes for the other 90.

## Your task

Fix `pricing` so bursts run at full speed:

1. Keep (or right-size) `requests.cpu` — requests are the **scheduler's**
   input (M5) and your capacity math; they cause no throttling and must stay.
2. Deal with the limit: raise it to real burst headroom (≥ `500m`) — or
   remove it entirely and let bursts use idle node CPU.
3. Prove it: re-read `cpu.stat` after a minute — `nr_throttled` should stop
   climbing.
