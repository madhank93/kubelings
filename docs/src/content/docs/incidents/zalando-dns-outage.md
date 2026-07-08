---
title: "Zalando: total DNS outage from ndots amplification"
description: "[REAL] Jan 2019 — retry traffic × no DNS caching × ndots:5 OOMKilled every CoreDNS pod at once. Cited case study with a runnable Kubelings replay."
---

> **[REAL] incident** — cited from Zalando's public postmortem:
> [Jan 2019 DNS outage](https://github.com/zalando-incubator/kubernetes-on-aws/blob/dev/docs/postmortems/jan-2019-dns-outage.md).
> **Runnable replay:** Kubelings lesson `incident-dns-ndots` (Module 4 — Networking).

## Situation

January 7, 2019. A downstream service in Zalando's fashion store starts timing
out. The aggregation layer retries; request volume spikes. The Node.js
applications resolve their upstream hostnames on **every** request — no DNS
caching — and connections were being recycled frequently, so lookup volume
tracked the retry storm 1:1.

## Blast radius

- **All CoreDNS pods OOMKilled simultaneously** → total cluster DNS failure.
- Fashion store web + outfit pages served high error rates **for over an hour**.
- Circuit breakers opened across downstream services.
- The monitoring stack itself needed cluster DNS → **automatic paging failed**,
  delaying human response.

## Root cause chain

1. **Trigger** — downstream timeout → retry amplification in the aggregation layer.
2. **Multiplier 1** — no application DNS caching: each request = fresh lookups.
3. **Multiplier 2** — Kubernetes' default `ndots:5` + search path turned every
   external-name lookup into **~10 real DNS queries** (each search-domain
   candidate, A and AAAA).
4. **Weak link** — CoreDNS ran with a `100Mi` memory limit; the amplified flood
   blew through it. All replicas saw the same load, so all died together.
5. **Blindfold** — monitoring shared fate with DNS; nobody was paged.

The outage wasn't one bug — it was multiplication across five reasonable-looking
decisions.

## Fix & prevention (what Zalando did)

- **Node-local DNS caching** (dnsmasq in front of CoreDNS; the modern equivalent
  is [NodeLocal DNSCache](https://kubernetes.io/docs/tasks/administer-cluster/nodelocaldns/)).
- Right-sized CoreDNS resources.
- **External, DNS-independent monitoring** so paging survives a DNS outage.
- Streamlined on-call escalation.

Per-workload defense you can apply today: set `dnsConfig.options` `ndots: "1"`
for pods that resolve external names in volume, or use absolute FQDNs with a
trailing dot (`payments.example.com.`).

## What it teaches

| Concept | Kubelings module |
|---|---|
| `ndots`/search-path amplification | M4 Networking — `incident-dns-ndots` (runnable) |
| Resource limits as a failure domain | M2 Workloads — `oomkill` |
| Shared-fate monitoring | M8 Observability & SRE |
