---
title: "Target: one network blip, five distributed systems down"
description: "[REAL] Jan 2019 — an OpenStack network upgrade made Kafka flap; logging sidecars woke at once, Docker daemons buckled, Kubernetes rescheduling registered 41,000 phantom nodes into Consul's gossip mesh, and Vault sealed itself. Days to recover."
---

> **[REAL] incident** — cited from Dan Woods (Target):
> [On Infrastructure at Scale: A Cascading Failure of Distributed Systems](https://medium.com/@daniel.p.woods/on-infrastructure-at-scale-a-cascading-failure-of-distributed-systems-7cff2a3cd2df).
> **Related Kubelings lessons:** `incident-target-cascade` (M9),
> `incident-monzo-cascade` (M9), `pattern-noisy-neighbor` (M5),
> `node-notready` (M8).

## Situation

January 2019. Target's internal platform (TAP) runs thousands of workloads
across heterogeneous infrastructure, with shared enterprise services — Kafka
for messaging and log/metric shipping, Consul for service discovery, Vault
for secrets. One evening, a routine upgrade to the **OpenStack network
subsystem** under the Kafka cluster is expected to cause a brief blip. It
disrupts connectivity for **hours** instead — and Kafka becomes
intermittently reachable.

## Blast radius

- The largest development Kubernetes cluster (~2,000 workloads) enters a
  self-perpetuating reschedule storm.
- **~41,000 phantom "nodes"** get registered into Consul's gossip mesh by
  short-lived pods that died before their apps even started.
- Consul latency becomes intolerable → the development **Vault seals
  itself** (it can't reach Consul), and TAP's deployment engine starts
  failing deployments.
- Recovery takes **days** of debugging. Production felt the same gossip
  poisoning but survived — its clusters were smaller and less densely packed.

## Root cause chain

1. **The blip** — OpenStack network upgrade makes Kafka intermittently
   unreachable for hours.
2. **The stampede** — every TAP workload ships with logging/metric sidecars.
   Kafka flapping made all of them "wake up" and retry **simultaneously**.
   Individually cheap; cumulatively enough to overload each node's shared
   Docker daemon.
3. **The death spiral** — overloaded Docker daemons made nodes report
   unhealthy; Kubernetes rescheduled their workloads onto still-healthy
   nodes, which then buckled under the same cumulative load. Repeat.
4. **The gossip poisoning** — each rescheduled pod carried a Consul agent
   sidecar that registered with the gossip mesh the moment it started —
   before the app ran. ~41,000 rapidly churning registrations flooded the
   mesh. Agents process only so many messages per loop, so "node gone"
   messages lagged "node joined" — phantom nodes expired and reappeared in
   waves.
5. **The dependents fall** — Vault sealed itself when Consul stopped
   answering in time; the deployment engine (service discovery, tokens,
   load-balancer config — all via Consul) began failing. The failure had now
   crossed five systems: OpenStack → Kafka → Kubernetes/Docker → Consul →
   Vault/deploys.

## Fix & prevention

- **Gossip encryption as a circuit breaker** — enabling Consul gossip
  encryption made the poisoned, unencrypted messages get rejected instantly;
  then a full redeploy pushed the config everywhere and terminated
  unencrypted stragglers.
- **Upgrade for scale** — Consul 1.2.3 shipped large-cluster gossip fixes;
  Target verified the upgrade with a controlled scale-up/ungraceful
  scale-down and watched nodes leave the mesh promptly.
- **"Smaller clusters, more of them"** — the outsized dev cluster cascaded;
  the smaller ones didn't, and neither did sparsely-packed production. Blast
  radius is a *sizing* decision.
- **Shared Docker daemon = shared fate** — the per-node daemon was the
  brittle choke point that converted many small retries into node failure.
- Per-workload sidecars were *kept*: had logging/metrics been a shared
  cluster service, the failure would have been worse and harder to fix.

## What it teaches

| Concept | Kubelings module |
|---|---|
| cascades cross system boundaries — think in graphs, not components | M9 — war stories |
| retry stampedes and thundering herds after a dependency flaps | M9 — `incident-monzo-cascade` |
| cumulative noisy neighbors take down the node, not the pod | M5 — `pattern-noisy-neighbor` |
| NotReady nodes + rescheduling can amplify, not heal | M8 — `node-notready` |
