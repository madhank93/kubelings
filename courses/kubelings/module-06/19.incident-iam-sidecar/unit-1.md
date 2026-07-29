---
kind: unit
title: "Incident file — the identity sidecar that added a zero to latency (Adevinta)"
name: incident-iam-sidecar-unit
---


> **Incident file (guided reading).** No lab: reproducing this needs a cloud IAM
> control plane and a metadata service on a link-local address. What transfers is
> the model — *how a pod gets cloud credentials, and what sits in the request
> path to make that happen* — which is now standard interview and design ground.
>
> Sources:
> [Adevinta — Kubernetes made my latency 10× higher](https://srvaroa.github.io/kubernetes/migration/latency/dns/java/aws/microservices/2019/10/22/kubernetes-added-a-0-to-my-latency.html) ·
> [Airbnb — 10 more weird ways to blow up your Kubernetes (kube2iam)](https://www.youtube.com/watch?v=4CT0cI62YHk)

## What happened

Adevinta moved a service to Kubernetes. Same code, same instance types, same
region — and p99 latency went up by roughly an order of magnitude. The traces
blamed AWS SDK calls. The AWS API was fine.

The pod was not talking to AWS directly. It was talking to **KIAM** (the same
family as **kube2iam**): an agent that intercepts requests to the EC2 metadata
address `169.254.169.254`, checks which pod the connection came from, assumes the
role that pod is annotated for, and hands back temporary credentials.

Three things compounded:

1. **Every credential fetch became a round trip through another pod** — a
   DaemonSet agent, subject to its own CPU limits, GC pauses, and cold start.
2. **DNS amplified it.** The JVM's SDK resolved names per call, and each lookup
   walked the `ndots:5` search-path ladder (M4 `incident-dns-ndots`) — several
   round trips per query — before every AWS request.
3. **The JVM cached nothing useful.** Java's default DNS TTL and the SDK's
   credential refresh conspired to re-do the expensive part far more often than
   anyone intended.

None of these show up as an error. They show up as a percentile.

## Why this failure mode is structural

Outside Kubernetes, a VM has *one* identity: the instance profile. Inside
Kubernetes, many pods with different privilege needs share a node — so something
must **de-multiplex identity per pod**. Every mechanism that does this inserts
itself into a hot path:

| Mechanism | How the pod gets credentials | What's in the path |
|---|---|---|
| Node instance profile | pod inherits the node's role | nothing — and every pod is equally privileged (the anti-pattern) |
| kube2iam / KIAM | agent hijacks `169.254.169.254`, assumes role per pod | a DaemonSet pod, iptables redirect, STS call |
| **IRSA** (EKS) / **Workload Identity** (GKE) / **Workload Identity Federation** (AKS) | kubelet projects a signed **ServiceAccount token**; the SDK exchanges it with STS directly | the projected token file + one STS exchange, cached by the SDK |
| SPIFFE/SPIRE, cert-manager + mTLS | workload identity documents / short-lived certs | agent socket, cert rotation |

The industry's answer to Adevinta's incident is the third row: **the projected
ServiceAccount token** (M6 `serviceaccount-tokens` — audience-scoped, short-TTL,
auto-rotated). It removes the interception hop entirely; the SDK does one
`AssumeRoleWithWebIdentity` and caches the result until expiry. kube2iam and KIAM
are both effectively retired because of exactly this.

```sh
# The modern shape, visible in any EKS/GKE cluster:
kubectl get sa <name> -o jsonpath='{.metadata.annotations}'      # role/identity binding
kubectl get pod <pod> -o jsonpath='{.spec.volumes[?(@.projected)].projected.sources}'
kubectl exec <pod> -- cat /var/run/secrets/eks.amazonaws.com/serviceaccount/token | cut -c1-20
```

## Concept checks

- The service was "just slower", with no errors and no restarts. Which signal
  would have found it in minutes? (Latency *attributed by dependency* — a span
  around credential acquisition, or simply an agent-side histogram. Percentiles
  on the app alone tell you that something is slow, never what.)
- Why does an identity agent make the tail worse specifically? (It's a shared,
  per-node resource with its own limits and queue — so its bad minute becomes
  every pod's bad minute, and contention shows in p99 long before p50.)
- Your pods use IRSA and calls still slow down under load. Where do you look
  first? (Token *audience/expiry* refresh storms, STS throttling, and — as
  always — DNS: the SDK still has to resolve `sts.<region>.amazonaws.com`.)

## What the industry took from it

- **Prefer native pod identity** (IRSA / Workload Identity) over metadata
  interception. Fewer hops, no privileged DaemonSet, and no iptables rule
  rewriting a link-local address underneath your app.
- **Block the metadata endpoint anyway.** A pod that can reach
  `169.254.169.254` can often steal the *node's* role — the escalation path in
  M6 `incident-cryptominer`, and one NetworkPolicy away from closed
  (M6 `egress-lockdown`).
- **Measure the credential path.** Add a span or metric around credential
  acquisition; it is the one dependency every cloud-touching pod has and almost
  nobody graphs.
- **Migrations change the *shape* of a call, not just its host.** "Same code,
  same region" hides new hops: identity agents, service mesh sidecars, CoreDNS,
  conntrack. When latency changes after a lift-and-shift, enumerate the hops that
  are new rather than re-profiling the application.

*No check — study, then advance.*

<!-- d2:creds -->
```text
 ┌──────────────────────┐
 │SDK needs credentials │
 │                      │
 └──────────────────────┘
            │            
     169.254.169.254     
            │            
            ▼            
  ┌───────────────────┐  
  │metadata agent hop │  
  │                   │  
  └───────────────────┘  
            │            
    DNS ladder + STS     
            │            
            ▼            
    ┌──────────────┐     
    │p99 10x worse │     
    │              │     
    └──────────────┘     
```
<!-- /d2:creds -->
