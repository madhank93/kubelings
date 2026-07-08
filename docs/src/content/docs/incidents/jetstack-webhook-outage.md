---
title: "Jetstack: the webhook that held the cluster hostage"
description: "[REAL] 2019 — a validating webhook with failurePolicy=Fail lost its backing pods on GKE; the API server rejected every write, including the recovery. Runnable Kubelings replay."
---

> **[REAL] incident** — cited from Jetstack's public postmortem:
> [GKE webhook outage](https://blog.jetstack.io/blog/gke-webhook-outage).
> **Runnable replay:** Kubelings lesson `incident-webhook-outage` (Module 6 — Security).

## Situation

During routine cluster operations on GKE, a Jetstack-managed cluster's API
server started refusing writes. Not some writes — effectively **all** of them:
deploys, fixes, scaling, the works.

## Blast radius

- Every mutating API request rejected while the webhook was unreachable.
- Recovery actions themselves are API writes — the failure mode **blocked its
  own fix**, the operator lockout pattern (compare OpenAI, Module 9).

## Root cause chain

1. **The mechanism** — admission webhooks sit *inside* the API server's write
   path: every matching request is sent to a service for verdict before
   persistence.
2. **The config** — a validating webhook was registered with
   `failurePolicy: Fail` ("if you can't reach me, reject the request") and a
   **broad scope** (matching far more resources than it needed to see).
3. **The trigger** — the webhook's backing pods became unavailable.
4. **The multiplication** — broad scope × Fail policy × dead backend =
   cluster-wide write outage. The webhook didn't do anything; its *absence* did.

## Fix & prevention

- **Scope tightly**: `namespaceSelector`/`objectSelector` and explicit rules so
  the webhook only sees what it must — never `*/*` with Fail.
- **Choose failurePolicy per consequence**: `Fail` only when a missed check is
  worse than a blocked cluster (rare); `Ignore` + alerting otherwise.
- **Exempt the escape hatches**: kube-system and the webhook's own namespace
  must never be gated by it, or recovery deadlocks.
- Run webhook backends like control-plane components: multiple replicas, PDBs,
  priority — they are availability-critical by construction.

## What it teaches

| Concept | Kubelings module |
|---|---|
| failurePolicy & scoping | M6 — `incident-webhook-outage` (runnable) |
| admission chain position | M7 — `control-plane-tour` |
| fail-safe vs fail-coupled | M9 — `incident-openai-cascade` |
