---
title: "Monzo: the cascade that stopped a bank"
description: "[REAL] Oct 2017 — an etcd change confused the service mesh, empty endpoints crashed clients, and the feedback loop stopped card payments for ~1.5 hours. Kubelings capstone reading."
---

> **[REAL] incident** — cited from Monzo's public incident thread:
> [Current-account payments may fail — major outage (27 Oct 2017)](https://community.monzo.com/t/resolved-current-account-payments-may-fail-major-outage-27-10-2017/26296/95),
> plus the follow-up KubeCon talk *Anatomy of a Production Kubernetes Outage*.
> **Kubelings capstone reading:** lesson `incident-monzo-cascade` (Module 9 — War Stories).

## Situation

27 October 2017. Monzo is a bank; for about 90 minutes, current-account
payments failed. Real people, real card declines. The trigger was a routine
infrastructure change; the outage was five familiar mechanisms composing.

## Blast radius

- Card payments and core banking flows degraded/down for ~1.5 hours.
- National-press-level visibility — blast radius measured in customers at
  tills, not pods.

## Root cause chain

1. **A routine change touched etcd** — the shared source of truth that both
   Kubernetes *and* their service mesh (linkerd) watched.
2. **The mesh got a bad view** — service discovery data became wrong; healthy
   services appeared to have **no endpoints**.
3. **Empty endpoints became crashes** — a client hit a code path that didn't
   expect an empty endpoint list: null-pointer exception, crash, not graceful
   degradation.
4. **The feedback loop** — crashing clients → endpoint churn → more bad
   discovery data → more crashes. The platform's healing machinery (restart,
   re-register, re-discover) *amplified* the fault.
5. **Payments sat downstream** — so the cascade surfaced as "payments may fail."

## Fix & prevention (what the industry adopted)

- **Graceful degradation over crashing**: empty/failed dependency = handled
  error + backoff + circuit breaker, never an unhandled exception.
- **Map shared-fate dependencies** before the incident — etcd underpinned two
  systems that then failed in correlated ways.
- **Damp the loops**: jitter, backoff, rate limits, circuit breakers so
  recovery converges instead of diverging.
- **Bulkheads** between infrastructure change and business-critical flows.

## What it teaches

| Concept | Kubelings module |
|---|---|
| the full cascade (guided study) | M9 — `incident-monzo-cascade` (reading) |
| etcd as shared truth | M7 — `etcd-truth` |
| empty endpoints / discovery | M1, M4 |
| reconciliation feedback loops | M7 — `reconcile-loop` |
