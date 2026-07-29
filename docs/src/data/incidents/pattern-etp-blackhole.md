---
title: "Pattern: externalTrafficPolicy Local blackholes nodes"
description: "[PATTERN] Synthetic composite — a Service that preserves the client IP silently drops traffic on every node that hosts no pod; drains, scale-downs and pinned placement all trigger it."
---

> **[PATTERN] scenario** — a synthetic composite of a failure mode reported
> across many production clusters. **No specific company**; details are
> representative, not cited. (Real, cited incidents are marked `[REAL]` in the
> [Incident Library](/catalog/).)

## Situation

A Service is created with `externalTrafficPolicy: Local` because the application
needs the real client IP — fraud scoring, rate limiting, access logs. Traffic
arrives through a load balancer (or a NodePort, or an on-prem VIP) spread across
nodes. Some fraction of requests fails with no server-side trace at all: no 502
from an ingress, no pod log, no connection reset you can attribute.

## Root cause chain

1. **`Local` means "deliver only to endpoints on this node".** There is no second
   hop, which is exactly why the source address survives.
2. **A node without a local endpoint therefore drops the packet.** Not an error,
   not a redirect — silence.
3. **Placement decides which nodes those are**, and placement drifts: a
   `nodeSelector`, bin-packing, an HPA scaling 6 → 2, or a `kubectl drain` that
   evicts the last pod on a node still in the load balancer's rotation.
4. **The health signal that should catch it is often mis-wired.** For
   `type: LoadBalancer` + `Local`, Kubernetes allocates a `healthCheckNodePort`
   that answers only while the node hosts a ready pod. A load balancer
   health-checking the app port instead — or a NodePort/VIP setup with no such
   check — keeps the blackhole in rotation.

## `Cluster` vs `Local`

| | `Cluster` (default) | `Local` |
|---|---|---|
| Node with no local endpoint | forwards to a pod elsewhere | **drops** |
| Client IP | SNAT'd to the node | **preserved** |
| Extra hop | yes (latency, cross-AZ cost) | no |
| Balance | even across pods | per-node share, skewed by placement |

## Fix & prevention

- Give **every node in rotation a local endpoint**: a DaemonSet, or a topology
  spread constraint plus enough replicas. On a cluster with tainted control-plane
  nodes, set `nodeTaintsPolicy: Honor` so unschedulable nodes don't count as
  empty domains — and pair a hard spread with `maxSurge: 0` so rollouts don't
  deadlock.
- **Point the load balancer at `healthCheckNodePort`**, and alert on it per node.
- **Re-check the placement story at every scale-down and drain** — the pairing of
  `Local` with placement is one decision, not two.
- If you don't need the client IP, use `Cluster` and buy none of this.

## What it teaches

| Concept | Kubelings module |
|---|---|
| externalTrafficPolicy, NodePort dataplane | M4 — `pattern-etp-blackhole` (drill) |
| Service types & kube-proxy forwarding | M4 — `nodeport-vs-clusterip`, `kube-proxy-dataplane` |
| spreading replicas across failure domains | M5 — `topology-spread`, `incident-same-node` |
| drains that remove the last endpoint | M8 — `node-maintenance` |
