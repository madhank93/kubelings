---
title: "Tinder: three failure modes on the way to 1,000 nodes"
description: "[REAL] 2019 — 250k DNS requests/sec, the conntrack DNAT race, and an ARP-cache outage: what Tinder's Kubernetes migration broke and how they fixed it."
---

> **[REAL] incident set** — cited from Tinder's first-party engineering post:
> [Tinder's move to Kubernetes](https://medium.com/tinder/tinders-move-to-kubernetes-cda2a6372f44)
> (Chris O'Brien, Chris Thomas, Jinyong Lee).
> **Related Kubelings lessons:** `incident-dns-ndots`, `incident-conntrack`,
> `kube-proxy-dataplane` (Module 4 — Networking).

## Situation

2019: Tinder migrates 200 services onto Kubernetes — 1,000 nodes, 15,000
pods, 48,000 containers. At that scale, three separate kernel-and-networking
failure modes surfaced that smaller clusters never see. One post, three
mini-postmortems.

## Failure 1 — DNS at 250,000 requests/second

Service discovery pushed cluster DNS to ~250k rps. Even after moving to
CoreDNS at absurd size (up to 1,000 pods eating 120 cores), lookups still
timed out intermittently. Volume alone wasn't the bug — it was the *exposer*
of failure 2. (The volume itself is the Zalando `ndots` amplification story:
every lookup multiplied by search-path candidates.)

## Failure 2 — the conntrack DNAT race

The intermittent timeouts traced to a **netfilter race condition** during
SNAT/DNAT — parallel UDP DNS packets colliding in connection tracking, some
silently dropped (the same kernel mechanism as the loveholidays/Preply
conntrack incidents, different bug). Mitigation: run **CoreDNS as a
DaemonSet** and point each node's pods at their local instance — DNS traffic
stops traversing SNAT/DNAT entirely. This is the architecture that later
became standard as **NodeLocal DNSCache**. Honest note in their post: the
race remained for *other* NATed traffic; they removed the biggest victim,
not the bug.

## Failure 3 — the ARP cache outage (January 8, 2019)

During scaling, **every node's ARP cache overflowed** (`gc_thresh3` — the
kernel's neighbor-table hard limit, default 1024). With 605 nodes on
Flannel's VXLAN overlay, the neighbor table outgrew the default; the kernel
started dropping entries, packets went nowhere, and services dropped. Fix:
raise the `gc_thresh*` kernel parameters and restart Flannel. The general
lesson is the sharpest one in the post: **node kernels ship with defaults
sized for a workstation, not for a node that talks to 15,000 pods.**
Conntrack limits, ARP tables, file descriptors, ephemeral ports — every
Kubernetes-at-scale postmortem features one of these ceilings.

## Bonus failure — keepalive pinning

HTTP keep-alive connections stuck to the pods they first landed on, leaving
some pods saturated while siblings idled (the per-*connection*, not
per-request balancing you saw in `kube-proxy-dataplane`). Their fix — Envoy
sidecars with least-request balancing — is the road that leads to service
mesh.

## What it teaches

| Concept | Kubelings module |
|---|---|
| DNS amplification & volume | M4 — `incident-dns-ndots` (runnable) |
| conntrack mechanics & races | M4 — `incident-conntrack` (reading) |
| why per-connection balancing pins | M4 — `kube-proxy-dataplane` (reading) |
| invisible node-level ceilings | M8 — `pattern-disk-pressure`, `incident-node-oom` |
