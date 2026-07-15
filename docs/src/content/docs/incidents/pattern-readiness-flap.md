---
title: "Pattern: readiness flapping"
description: "[PATTERN] Synthetic composite — hair-trigger readiness probes cycle healthy pods in and out of the endpoint pool, surfacing as intermittent 502s."
---

> **[PATTERN] scenario** — a synthetic composite of a failure mode reported
> across many production clusters. **No specific company**; details are
> representative, not cited. (Real, cited incidents are marked `[REAL]` in the
> [Incident Library](/catalog/).)

## Situation

Users report intermittent 502s. Nothing is down: no restarts, no OOMKills, no
failed deploys. But `kubectl get pods -w` shows `READY` flickering `1/1 → 0/1
→ 1/1` every few seconds, and the Service's endpoint list churns with it.
Requests in flight when a pod leaves the pool get 502s; monitoring shows an
error *rate*, never an outage.

## Root cause

A readiness probe tuned to a hair trigger — typically `periodSeconds: 1`,
`failureThreshold: 1` — against a health endpoint that occasionally runs slow
under load (GC pause, connection-pool wait, noisy neighbor). One transient
blip fails one probe, and one failed probe is the whole eviction budget: the
endpoints controller removes the pod immediately, re-adds it a second later,
forever.

The app is healthy. The *probe* is converting normal latency variance into
routing churn.

## Diagnosis

```sh
# flapping READY column:
kubectl get pods -l app=<name> -w
# endpoint churn, the smoking gun:
kubectl get endpoints <svc> -w
# the trigger-happy settings:
kubectl get deploy <name> -o jsonpath='{.spec.template.spec.containers[0].readinessProbe}'
# probe failures in events:
kubectl describe pod -l app=<name> | grep -c "Readiness probe failed"
```

## Fix

Give the probe tolerance — it takes *consecutive* failures to evict:

```yaml
readinessProbe:
  periodSeconds: 5        # was 1
  failureThreshold: 3     # was 1
  timeoutSeconds: 2
```

Now unreadiness requires ~15 s of *sustained* failure — a real problem — while
isolated blips never string 3 in a row.

## Prevention

- Readiness ≠ liveness: a twitchy readiness probe causes 502 churn, a twitchy
  liveness probe causes restart storms. Tune each for its failure cost.
- Baseline: `periodSeconds: 5–10`, `failureThreshold: 3`; use a
  `startupProbe` for slow boots instead of shrinking readiness settings.
- Keep the probed path cheap and dependency-free — probing through your
  database makes its blips your outage.
- Alert on endpoint-churn rate; restart counters are blind to this pattern.

## What it teaches

| Concept | Kubelings module |
|---|---|
| Probe tuning, endpoint pool mechanics | M2 Workloads (`pattern-readiness-flap`) |
| Services & endpoint propagation | M4 Networking |
