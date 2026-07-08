---
title: "Conntrack exhaustion: the table nobody knew existed"
description: "[REAL] loveholidays & Preply — the Linux connection-tracking table filled up and silently dropped production traffic and DNS. Kubelings reading with both citations."
---

> **[REAL] incidents** — two independent, cited write-ups of the same failure:
> [Preply / deploy.live — Kubernetes networking problems due to conntrack](https://deploy.live/blog/kubernetes-networking-problems-due-to-the-conntrack/)
> (and loveholidays' matching account).
> **Kubelings reading:** lesson `incident-conntrack` (Module 4 — Networking).

## Situation

Two different companies, same story: intermittent timeouts and weird DNS
failures in production with *nothing* apparently wrong — pods healthy, services
correct, CPU fine. The failing layer was one almost no application team
watches: **netfilter's connection-tracking table** on each node.

## Blast radius

- Silently dropped new connections at the node level — timeouts with no error
  logs anywhere in the application stack.
- DNS especially hurt: high-volume, short-lived UDP flows churn conntrack
  entries fastest (see the ndots amplifier — Zalando — for why DNS volume is
  always higher than you think).

## Root cause chain

1. **The mechanism** — kube-proxy's NAT (Services!) relies on conntrack; every
   flow through a ClusterIP occupies a table entry until it expires.
2. **The ceiling** — `nf_conntrack_max` is finite per node. When the table is
   full, the kernel **drops new connections** — no RST, no log line in your
   app, just `insert_failed` counters ticking on the node.
3. **The trigger** — traffic growth, connection-churn-heavy workloads, or many
   short-lived UDP flows (DNS) on dense nodes crossed the ceiling.
4. **The invisibility** — nothing in `kubectl` shows it. Node-level metric or
   nothing.

## Fix & prevention

- Raise `nf_conntrack_max` (kube-proxy's `--conntrack-max-per-core` / config) —
  memory cost is modest; the default is sized for another decade.
- **Monitor `conntrack_entries` vs limit and `insert_failed`** per node —
  this is the entire early-warning system.
- Reduce churn: connection reuse/keep-alives, NodeLocal DNSCache for the DNS
  flood component.

## What it teaches

| Concept | Kubelings module |
|---|---|
| conntrack & kube-proxy NAT | M4 — `incident-conntrack` (reading) |
| DNS volume amplification | M4 — `incident-dns-ndots` |
| node-level invisible limits | M8 — `pattern-disk-pressure`, `incident-node-oom` |
