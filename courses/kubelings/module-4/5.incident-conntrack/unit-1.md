---
kind: unit
title: "Incident file — conntrack: the invisible table that fills up"
name: incident-conntrack-unit
---


> **Incident file (guided reading).** Two real, cited production incidents.
> Conntrack exhaustion can't be reproduced safely on a shared laptop kernel —
> kind nodes share your machine's conntrack table — so this lesson trains the
> *recognition*, which is 90% of surviving it.
>
> Sources:
> [loveholidays — Kubernetes networking problems due to conntrack](https://deploy.live/blog/kubernetes-networking-problems-due-to-the-conntrack/) ·
> [Preply — DNS postmortem #1](https://medium.com/preply-engineering/dns-postmortem-e169efd45afd)

## What conntrack is

Every NAT'd connection through a Linux box gets a row in the kernel's
**connection-tracking table**: who talked to whom, ports, state, timeout. Kubernetes
*runs* on NAT — every Service VIP hop, every SNAT'd egress is a conntrack entry
on that node.

The table has a fixed size (`nf_conntrack_max`, often 128k–256k). While it has
room, it's invisible. When it fills:

```
nf_conntrack: table full, dropping packet
```

New connections silently fail. Existing ones continue. **The node looks healthy;
only *new* traffic dies** — the exact inverse of what your health checks test.

## Incident 1 — loveholidays: the table full at peak

GKE cluster, HAProxy ingress, traffic peak. Symptoms: rising 5xx, but pods
healthy, CPU fine, memory fine — every dashboard green except customer errors.
The kernel log line above was the entire diagnosis; everything else was noise.

Root cause chain: high connection rate × short-lived connections × conntrack
entries lingering in `TIME_WAIT` → table full → new connections dropped at the
node, before any pod saw them.

Fixes: raised `nf_conntrack_max`, tuned timeout sysctls, moved the busiest
path to fewer, longer-lived connections (keep-alive).

## Incident 2 — Preply: the DNS race nobody could see

Different shape, same table. Preply saw intermittent 5-second DNS stalls — the
infamous pattern anyone who runs Kubernetes at scale eventually meets.

Mechanism (worth understanding precisely): glibc sends A and AAAA lookups **in
parallel, over the same UDP socket**, through DNAT to the DNS Service VIP.
Conntrack insertion for the two packets **races**; the loser's entry is dropped;
one query vanishes; the resolver waits its full timeout (5s) before retrying.
It's a kernel race (netfilter), so it strikes probabilistically — worse under
load, gone in dev, unkillable by scaling DNS pods.

Mitigations the industry converged on:

- **NodeLocal DNSCache** — pods talk to a local daemon, no DNAT, no race.
- `single-request-reopen` in resolv.conf options (serialize A/AAAA) — the
  Preply-era workaround.
- Fewer lookups altogether: the `ndots` fix you shipped in the
  [previous lesson](../1.incident-dns-ndots/) multiplies every gain here.

## The recognition kit (memorize)

| Signal | Smell |
|---|---|
| "table full, dropping packet" in `dmesg` | conntrack exhaustion — raise max, tune timeouts |
| Intermittent exactly-5s DNS stalls | the UDP race — NodeLocal DNSCache |
| New connections fail, existing fine, dashboards green | connection-level, not app-level: think conntrack, SNAT ports, ephemeral range |
| Node-level metrics: `node_nf_conntrack_entries` near `_max` | you're weeks from the incident — act now |

## Why this matters for the curriculum

You now know three DNS/network failure layers that *compose*: ndots amplifies
query volume → volume fills conntrack → conntrack races corrupt what's left.
Zalando, Preply, and loveholidays are one story told three ways: **the platform's
invisible defaults are load-bearing.** Module 7 shows you the machinery itself.

*No check for this lesson — mark it done by moving on. The next one has you back
in the terminal.*
