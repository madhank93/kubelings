---
title: "Exponea: how we failed to integrate Istio"
description: "[REAL] 2019 — a mesh rollout abandoned after the sidecar broke Jobs, StatefulSets, gRPC balancing and graceful shutdown in turn. Kubelings incident file."
---

> **[REAL] write-up** — cited from Exponea's public account:
> [Sailing with Istio through the shallow water](https://medium.com/@jakubkulich/sailing-with-the-istio-through-the-shallow-water-8ae81668381e).
> **Reading lesson:** Kubelings `incident-istio-integration`
> (Module 10 — Platform Engineering).

## Situation

Exponea (GKE, microservices) adopted Istio for traffic insight, mTLS and moving
retries/timeouts out of application code. Managed Istio on GKE turned out to be
configurable only for mTLS mode, so they ran it themselves — and then met the
sidecar, workload class by workload class.

## Blast radius

Not an outage report: an adoption that was **abandoned**. The cost is worth
cataloguing precisely because it is the cost every mesh evaluation pays.

## What broke

1. **Port naming became load-bearing** — Istio infers protocol from the port
   *name*; wrongly named ports changed traffic handling and broke startup.
2. **Jobs never completed** — the app container exits, the proxy does not, so the
   pod never reaches Completed and anything waiting on the Job hangs.
3. **StatefulSets misbehaved** — service discovery failed to track pod IP changes
   on recreation, routing to addresses that no longer existed.
4. **gRPC client-side load balancing conflicted** with the proxy's balancing;
   headless Services and the mesh disagreed about whose job it was.
5. **Graceful shutdown regressed** — sidecars exited on SIGTERM immediately,
   truncating work that needed the full grace period.
6. **No upgrade story** — fleet-wide sidecar version rollout plus unsupported
   multi-tenancy made production adoption a project, not a step.

## The single fact underneath

A mesh puts **a second process in every pod and a second hop in every call**.
Every symptom above is a consequence: two lifecycles in one pod, traffic that is
parsed rather than forwarded, discovery/balancing ownership moving, and a new
control plane to operate.

## What changed since (and what didn't)

- Kubernetes **sidecar containers** (1.29+) fix the Job-never-completes class
  outright: the proxy is an init container with `restartPolicy: Always` and is
  terminated when the main container exits.
- `holdApplicationUntilProxyStarts` fixes the startup race.
- **Ambient mode** removes the per-pod sidecar entirely for many use cases.
- Unchanged: the mesh is a control plane you operate, its injector is an
  admission webhook in your critical path, and every request pays proxy CPU.

## Adoption guidance

- Roll out **workload class by workload class** — stateless HTTP first; Jobs,
  StatefulSets and client-side-balanced gRPC last and deliberately.
- **Exclude by default, include on purpose** (injection labels are your blast
  radius control).
- **Budget the tax first**: proxy CPU/memory × fleet size, plus added p99.
- Reach for smaller tools when they suffice: Gateway API for north-south,
  NetworkPolicy for segmentation, OTel for traces.

## What it teaches

| Concept | Kubelings module |
|---|---|
| service mesh trade-offs & adoption strategy | M10 — `incident-istio-integration` (reading) |
| sidecar lifecycle vs Job completion | M2 — `incident-job-restartpolicy`, `multi-container-patterns` |
| admission webhooks as a failure domain | M6 — `incident-webhook-outage` |
| graceful shutdown & endpoint removal | M4 — `incident-graceful-shutdown` |
