---
title: "OpenAI: locked out of the control plane"
description: "[REAL] Dec 2024 — a fleet-wide telemetry rollout overwhelmed every API server at once; DNS caching hid it until it was everywhere, then blocked the fix. Kubelings capstone reading."
---

> **[REAL] incident** — cited from OpenAI's public postmortem:
> [API, ChatGPT & Sora facing issues (11 Dec 2024)](https://status.openai.com/incidents/ctrsv3lwd797).
> **Kubelings capstone reading:** lesson `incident-openai-cascade` (Module 9 — War Stories).

## Situation

11 December 2024. ChatGPT, the API, and Sora down for roughly four hours. The
change that did it was a **telemetry service** meant to *improve* control-plane
observability, deployed to every cluster in the fleet within a short window.

## Blast radius

- Kubernetes API servers saturated fleet-wide — controllers, kubectl, watches
  all degraded or dead.
- Data-plane workloads kept running but **DNS-based service discovery** decayed
  as cached answers expired → user-facing outage.
- Operators **locked out**: removing the bad deployment required the very API
  that was down.

## Root cause chain

1. **Aggregate load** — the agent's per-node API cost was individually
   reasonable; per-node cost × the largest clusters' node counts crossed the
   API servers' capacity. Tested, but not at top-of-fleet scale.
2. **Hidden coupling** — runtime request-serving depended on DNS records
   resolved from cluster state: a control-plane outage reached the data plane.
3. **The cache fuse** — DNS caches (~20 min) kept everything looking healthy
   until the rollout was *everywhere*, then expired fleet-wide on a timer.
4. **The lockout** — the fix was one `kubectl delete`; the door it needed was
   the thing that was down. Recovery meant shrinking clusters, blocking
   non-admin API traffic, and scaling API servers just to win back enough
   headroom to issue it.

## Fix & prevention (what the postmortem committed to)

- **Staged rollouts with real bake time** — longer than every cache TTL the
  blast radius touches; "looks healthy" is not a gate when symptoms are
  time-delayed.
- **Decouple the data plane** from control-plane availability for runtime
  serving.
- **Break-glass access** to the control plane that doesn't compete with
  normal API traffic — and is exercised.
- Protect API servers from their tenants: priority & fairness, per-client
  limits, load tests sized to the *largest* cluster.

## What it teaches

| Concept | Kubelings module |
|---|---|
| the full cascade (guided study) | M9 — `incident-openai-cascade` (reading) |
| control plane vs data plane | M7 — `control-plane-tour` |
| webhook-style fail-coupled design | M6 — `incident-webhook-outage` |
| DNS as load-bearing infrastructure | M4 — `incident-dns-ndots` |
