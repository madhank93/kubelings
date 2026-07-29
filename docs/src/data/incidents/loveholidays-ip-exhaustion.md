---
title: "loveholidays: the cluster that ran out of IP addresses"
description: "[REAL] 2020 — a /16 pod range that looked enormous capped the cluster at 256 nodes; pods stuck Pending and the autoscaler refused to scale. Kubelings incident file."
---

> **[REAL] incident** — cited from loveholidays' public write-up:
> [When GKE ran out of IP addresses](https://deploy.live/blog/when-gke-ran-out-of-ip-addresses/).
> **Reading lesson:** Kubelings `incident-ip-exhaustion` (Module 4 — Networking).

## Situation

loveholidays ran GKE in VPC-native mode, where pods take real VPC addresses from
a subnet's secondary range. Pods began sitting **Pending**; roughly half of newly
created pods never reached Running, and the cluster autoscaler declined to add
nodes with "IP space of subnetwork is exhausted".

## Blast radius

- Deployments hung mid-rollout; capacity could not be added during the incident.
- The ceiling was structural — no manifest change could clear it.

## Root cause chain

1. **GKE reserves a block per node, not an address per pod** — sized at *double*
   max-pods, so addresses aren't immediately reused (a reused pod IP delivers
   traffic to the wrong workload).
2. **Default max-pods 110 ⇒ a `/24` (256 addresses) reserved per node.**
3. **A `/16` secondary range therefore caps the cluster at 256 nodes**, not the
   ~595 the team had assumed from 65,536 ÷ 110.
4. **The symptom pointed at the wrong layer.** Pending pods look like a scheduler
   problem; the failure is at pod-sandbox creation, in the CNI's IPAM
   (`failed to allocate for range 0: no IP addresses available`), while the
   autoscaler's matching failure appears in a completely different log.

## Fix & prevention

- **Most effective:** lower **max-pods per node pool** for pools that don't need
  110 pods — recovered ~30% of the range at no cost.
- **Expand the secondary range** where the provider allows it (e.g. `/24` → `/23`
  on additional ranges).
- **Bigger nodes, fewer of them** — one reservation each — traded against the
  blast radius of losing a node.
- **Alert on subnet utilisation**, not pod counts; a 70% threshold converts this
  outage into a ticket.
- Do the arithmetic **before cluster creation** and write it down: the ceiling is
  `range size ÷ per-node block`, and it cannot be raised later.

## Same failure, other clouds

| Environment | Where the ceiling hides |
|---|---|
| EKS + VPC CNI | ENIs × IPs-per-ENI **per instance type** (a `t3.small` caps at 11 pods) |
| AKS Azure CNI | subnet must hold nodes × (max-pods + 1) |
| kubeadm + routed CNI | `--pod-network-cidr` split into per-node `/24`s — a `/16` is 256 nodes, permanently |

## What it teaches

| Concept | Kubelings module |
|---|---|
| pod IPAM, per-node CIDR, cluster sizing | M4 — `incident-ip-exhaustion` (reading) |
| CNI anatomy and sandbox creation | M4 — `cni-basics` |
| stale flow state when IPs are reused | M4 — `incident-conntrack` |
| Pending-pod triage | M8 — `events-forensics`, `quota-exhausted` |
