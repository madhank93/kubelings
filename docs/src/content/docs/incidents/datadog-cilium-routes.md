---
title: "Datadog: the OS update that unplugged five regions"
description: "[REAL] Mar 2023 — an auto-applied systemd update restarted systemd-networkd, which deleted Cilium's routes on tens of thousands of nodes at once. Kubelings reading."
---

> **[REAL] incident** — cited from Datadog's public postmortem:
> [2023-03-08 infrastructure connectivity issue](https://www.datadoghq.com/blog/2023-03-08-multiregion-infrastructure-connectivity-issue/).
> **Kubelings reading:** lesson `incident-datadog-cilium` (Module 8 — Observability & SRE).

## Situation

8 March 2023, ~06:00 UTC. Datadog — whose product *is* observability — went
dark for more than a day. Tens of thousands of nodes across **five regions and
multiple cloud providers** dropped off the network within about an hour of
each other.

## Blast radius

- Fleet-wide node connectivity loss: pod networking gone wherever the update
  landed.
- Mass NotReady → mass eviction attempts with nowhere to land; recovery itself
  became a capacity crisis.
- Status page and customer comms lagged — the observer shared fate with the
  observed.

## Root cause chain

1. **Trigger** — a routine **systemd security update**, delivered by the OS's
   automatic update channel on Ubuntu nodes.
2. **The interaction** — applying it restarted `systemd-networkd`, which
   removed routes it didn't own — **including the routes Cilium (the CNI) had
   installed** for pod traffic.
3. **The correlation** — auto-updates ran in the same overnight window
   fleet-wide. Regions and clouds were irrelevant as failure domains: the
   update channel cut across all of them.
4. **The recovery tax** — fixing routes was fast; rebooting/replacing compute
   at fleet scale while every workload rescheduled at once was the slow,
   days-long tail.

## Fix & prevention

- **Ring-deploy the OS layer**: canary nodes → health gate (CNI routes
  present? node Ready?) → staggered waves. An unstaggered update channel is a
  global simultaneous deploy you didn't schedule.
- **Immutable, versioned node images** instead of in-place auto-updates —
  CVE patches become reviewable, rollback-able releases.
- Know what owns the routes on your nodes; the networkd/Cilium interaction had
  a known config mitigation.
- **Out-of-band monitoring & status** so you can see and say what's happening.

## What it teaches

| Concept | Kubelings module |
|---|---|
| the OS under the cluster / CNI reality | M8 — `incident-datadog-cilium` (reading) |
| NotReady lifecycle at scale | M8 — `node-notready` |
| correlated failure domains | M9 — `incident-monzo-cascade` |
