---
title: "Adevinta: the identity sidecar that added a zero to latency"
description: "[REAL] 2019 — a lift-and-shift to Kubernetes made p99 10× worse; the new hop was the per-pod IAM agent intercepting the metadata service, amplified by DNS. Kubelings incident file."
---

> **[REAL] incident** — cited from the public write-up:
> [Kubernetes made my latency 10× higher](https://srvaroa.github.io/kubernetes/migration/latency/dns/java/aws/microservices/2019/10/22/kubernetes-added-a-0-to-my-latency.html).
> **Reading lesson:** Kubelings `incident-iam-sidecar` (Module 6 — Security).

## Situation

A service moved to Kubernetes on AWS. Same code, same region, same instance
types — and p99 latency rose by roughly an order of magnitude. Traces implicated
AWS SDK calls; the AWS APIs themselves were healthy.

## Blast radius

- No errors, no restarts, no failed deploys — only a percentile. The regression
  survived initial investigation precisely because nothing was "down".

## Root cause chain

1. **Pod identity was provided by KIAM/kube2iam**, agents that intercept requests
   to the EC2 metadata address `169.254.169.254`, identify the calling pod, and
   assume its role.
2. **Every credential fetch became a network round trip to another pod** — a
   DaemonSet with its own limits, queueing and cold starts, shared by everything
   on the node.
3. **DNS amplified it.** Per-call name resolution walked the `ndots:5` search-path
   ladder before each AWS request.
4. **JVM/SDK caching made the expensive path frequent** rather than rare.

Each hop was individually reasonable. Their product was a tail-latency
regression that no single component reported as a fault.

## Fix & prevention

- **Use native pod identity**: IRSA (EKS), Workload Identity (GKE), Workload
  Identity Federation (AKS). The kubelet projects a signed, audience-scoped
  ServiceAccount token; the SDK exchanges it with STS directly and caches the
  credentials — no interception hop, no privileged DaemonSet. kube2iam and KIAM
  are effectively retired because of this.
- **Block the metadata endpoint for pods anyway** — reaching it is a standard
  path to stealing the node's role.
- **Instrument the credential path** (a span or an agent-side histogram); it is
  the one dependency every cloud-touching pod shares and almost nobody graphs.
- **When latency changes after a migration, enumerate the new hops** — identity
  agents, mesh sidecars, CoreDNS, conntrack — before re-profiling the app.

## What it teaches

| Concept | Kubelings module |
|---|---|
| pod identity, projected SA tokens, IRSA | M6 — `incident-iam-sidecar` (reading), `serviceaccount-tokens` |
| metadata endpoint as an escalation path | M6 — `incident-cryptominer`, `egress-lockdown` |
| DNS search-path amplification | M4 — `incident-dns-ndots` |
| latency attribution by dependency | M8 — `otel-collector-pipeline`, `slo-errorbudget` |
