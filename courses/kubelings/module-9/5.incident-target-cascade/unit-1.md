---
kind: unit
title: "Incident file — the five-system cascade (Target, 2019)"
name: incident-target-cascade-unit
---


> **Capstone incident file (guided study).** No lab — the fault crosses
> OpenStack, Kafka, Docker, Kubernetes, Consul and Vault; what's reproducible
> here is the *cascade reasoning*. Read the postmortem itself — it's a
> masterclass in tracing failure across system boundaries.
>
> Source:
> [Dan Woods (Target) — "On Infrastructure at Scale: A Cascading Failure of Distributed Systems"](https://medium.com/@daniel.p.woods/on-infrastructure-at-scale-a-cascading-failure-of-distributed-systems-7cff2a3cd2df)
> · [full case study](https://kubelings.madhan.app/incidents/target-cascade/)

## What happened

January 2019. A routine upgrade to the OpenStack network subsystem under
Target's Kafka cluster was supposed to cause a brief blip. Instead it
disrupted connectivity for hours — and by the time the dust settled, a
~2,000-workload Kubernetes cluster was in a reschedule death spiral,
Consul's gossip mesh held **41,000 phantom nodes**, Vault had sealed itself,
and deployments were failing. Recovery took **days**.

## The chain

**Link 1 — the stampede.** Every workload shipped with logging/metric
sidecars feeding Kafka. When Kafka flapped, *all of them* woke and retried
at once. Each was individually cheap — cumulatively they overloaded the
shared Docker daemon on each node. The noisy-neighbor lesson (M5), but the
neighbor is *everyone at once*.

**Link 2 — the death spiral.** Loaded daemons made nodes report NotReady
(M8.4); Kubernetes rescheduled their pods onto healthy nodes, which then
buckled under the same cumulative load, and the cycle fed itself. The
scheduler doesn't know *why* nodes are failing — rescheduling into a
capacity-collapse amplifies it.

**Link 3 — the gossip poisoning.** Each pod carried a Consul agent sidecar
that registered with the mesh at pod start — before the app ran. The
reschedule storm registered ~41,000 short-lived phantom nodes. Consul agents
process a bounded number of gossip messages per loop, so departures lagged
arrivals and phantoms re-expired and re-joined in waves. The mesh was now
poisoned *independently of the original outage* — Kafka could recover;
Consul wouldn't.

**Link 4 — the dependents.** Vault sealed itself when Consul stopped
answering in time. The deployment engine — discovery, tokens, load-balancer
config, all via Consul — started failing. Five systems in, the incident no
longer resembles its root cause.

**Link 5 — the recovery lever.** After days: enable **gossip encryption**,
so poisoned unencrypted messages get rejected outright — a protocol-level
circuit breaker — then redeploy everything and upgrade Consul to a version
with large-cluster gossip fixes.

## Concept checks

- Production had the same poisoned gossip mesh but no cascade. Why? (Smaller,
  sparsely-packed clusters — the load never crossed the tipping point.
  "Smaller clusters, more of them" is a blast-radius decision, same
  conclusion Spotify reached — next lesson.)
- Where does *your* platform register pods into external systems before the
  app is healthy? What happens to that system under a reschedule storm?
- The sidecars caused the stampede — and the postmortem *defends* keeping
  them. Why? (Shared cluster-level logging would have been one bigger, harder
  failure. Per-workload sidecars fail per-workload.)

## What the industry took from it

- **Cascade thinking**: the fifth system down had never heard of OpenStack.
  Map the dependency graph, not just each component's health.
- **Retry stampedes are a design input** — anything that makes "all clients
  retry at once" possible needs jitter, backoff and admission control.
- **Shared per-node daemons are choke points**: many small overloads sum
  into node failure (Docker then; kubelet/containerd today).
- **Registration ≠ readiness**: register services when they can serve, not
  when the process starts.

*No check — study, then advance. Next: the war story where deleting every
cluster was survivable.*
