---
title: "Ravelin: 502s on every deploy"
description: "[REAL] — rolling updates produced bursts of 502s because pod termination and endpoint removal run in parallel, not in order. Runnable Kubelings replay."
---

> **[REAL] incident** — cited from Ravelin engineering (Phil Pearl):
> [Delays and pain with Kubernetes graceful shutdown / ingress](https://philpearl.github.io/post/k8s_ingress/).
> **Runnable replay:** Kubelings lesson `incident-graceful-shutdown` (Module 4 — Networking).

## Situation

Ravelin (fraud detection) noticed that **every rolling update** — the routine,
zero-downtime-by-design operation — produced a burst of 502s at the edge.
No crash, no bad release: the deploy mechanism itself was dropping requests.

## Blast radius

- Small error burst on **every single deploy**, cluster-wide pattern.
- Individually minor; multiplied by deploy frequency it's a permanent
  reliability tax — and it erodes trust in shipping.

## Root cause chain

1. **The assumption** — "Kubernetes removes the pod from load balancing, *then*
   stops it."
2. **The reality** — pod deletion fans out **in parallel**: the kubelet sends
   SIGTERM *at the same time* as the endpoint controller starts removing the
   pod from Endpoints and kube-proxy/ingress converge on that change.
3. **The race** — for a window of milliseconds-to-seconds, traffic still
   arrives at a pod that's already been told to die. If the app exits promptly
   on SIGTERM (being a good citizen!), it closes the door on in-flight and
   newly-routed requests → 502s.
4. **The irony** — the *faster* your app shuts down, the *worse* the burst.

## Fix & prevention (what the write-up landed on)

- A deliberately "dumb" **preStop sleep** (a few seconds): delays SIGTERM so
  endpoint removal wins the race — ugly, universal, effective.
- App side: on SIGTERM, **keep serving** while draining; stop accepting new
  work, finish in-flight, then exit within `terminationGracePeriodSeconds`.
- Edge side: retries for idempotent requests absorb the residue.

## What it teaches

| Concept | Kubelings module |
|---|---|
| termination vs endpoints race | M4 — `incident-graceful-shutdown` (runnable) |
| readiness as the traffic gate | M2 — `liveness-vs-readiness` |
| deploy-time availability | M2 — `rolling-update`, `blue-green-canary` |
