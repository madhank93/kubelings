---
title: "Moonlight: all replicas on the node that died"
description: "[REAL] 2019 — every pod of the website scheduled onto one host; that host failed. 100% loss with 'redundant' replicas. Runnable Kubelings replay."
---

> **[REAL] incident** — cited from Moonlight's public postmortem:
> [outage post-mortem](https://updates.moonlightwork.com/outage-post-mortem-87370).
> **Runnable replay:** Kubelings lesson `incident-same-node` (Module 5 — Scheduling & Placement).

## Situation

Moonlight (a developer hiring marketplace) ran its website as multiple replicas
on GKE — redundant, on paper. Then the site went fully down, repeatedly, over
several days of intermittent chaos.

## Blast radius

- Website hard down — 100% traffic loss despite multiple replicas.
- Repeated recurrences as the same scheduling shape kept reassembling itself.

## Root cause chain

1. **Trigger** — GKE nodes started failing (kernel panics on the underlying VMs).
2. **The silent setup** — the Kubernetes scheduler packs pods onto nodes with
   free resources; nothing told it the website's replicas should avoid each
   other. All replicas of the website landed on **the same node**.
3. **Correlation** — that node was one of the ones that panicked. Every replica
   died in the same instant — replica count 3, failure domains 1.
4. **Recurrence** — after recovery, the scheduler happily rebuilt the same
   stack-everything-together layout, so the next node failure repeated the show.

Replicas are only redundancy if they can't share a single point of failure.
Count ≠ spread.

## Fix & prevention (what Moonlight did)

- **Pod anti-affinity** so replicas of the same service repel each other across
  nodes (`podAntiAffinity` on the hostname topology key).
- The modern, simpler default for the same goal: **topologySpreadConstraints**
  (see Kubelings `topology-spread`).
- Platform-side: keep node auto-repair on and watch for correlated node
  failures — but treat *placement* as the app team's contract.

## What it teaches

| Concept | Kubelings module |
|---|---|
| anti-affinity / spreading | M5 — `incident-same-node` (runnable) |
| proportional spread | M5 — `topology-spread` |
| blast-radius thinking | M9 War Stories |
