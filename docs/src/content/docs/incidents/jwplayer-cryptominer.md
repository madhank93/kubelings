---
title: "JW Player: the cryptominer in the cluster"
description: "[REAL] 2018 — an internal ops tool exposed via public LoadBalancer with no auth; attackers used it to run a miner across the cluster. Runnable Kubelings replay."
---

> **[REAL] incident** — cited from JW Player's public write-up:
> [How a cryptocurrency miner made its way onto our internal Kubernetes clusters](https://medium.com/jw-player-engineering/how-a-cryptocurrency-miner-made-its-way-onto-our-internal-kubernetes-clusters-9b09c4704205).
> **Runnable replay:** Kubelings lesson `incident-cryptominer` (Module 6 — Security).

## Situation

Monitoring flagged unusual CPU saturation on JW Player's internal (staging/dev)
Kubernetes clusters. Investigation found a process nobody deployed: a Monero
miner, running happily alongside internal workloads.

## Blast radius

- Miner workload running across internal clusters — stolen compute, and an
  attacker with a foothold inside the cluster network.
- No customer data impact reported — but only because the intruder monetized
  CPU instead of moving laterally.

## Root cause chain

1. **The door** — an internal ops/visualization tool was exposed to the
   internet through a public **LoadBalancer Service with no authentication**.
2. **The capability** — the tool could execute commands in its host
   environment; anyone reaching the URL inherited that power.
3. **The payload** — attackers (scanning the internet for exactly this) used it
   to launch a miner.
4. **The defaults that helped** — permissive workload identity and no egress
   restrictions meant the miner could run, resolve its pool, and phone home
   unimpeded.

Nobody "hacked Kubernetes." A convenience tool plus a public IP was enough.

## Fix & prevention (what JW Player did / what it teaches)

- Removed the exposure; internal tools live behind VPN/ingress auth — a
  `LoadBalancer` Service is a **publication event**, review it like one.
- Audit standing exposure: every external IP in `kubectl get svc -A` needs an
  owner and a reason.
- Contain the next foothold: least-privilege ServiceAccounts
  (`serviceaccount-tokens`), hardened containers (`container-hardening`), and
  **egress lockdown** so a miner can't reach its pool (`egress-lockdown`).
- Detection: CPU anomalies found this one; runtime security (Falco-class
  tooling, see `control-plane-hardening`) finds it faster.

## What it teaches

| Concept | Kubelings module |
|---|---|
| exposure + workload response | M6 — `incident-cryptominer` (runnable) |
| egress as the kill switch | M6 — `egress-lockdown` |
| identity & token hygiene | M6 — `serviceaccount-tokens` |
