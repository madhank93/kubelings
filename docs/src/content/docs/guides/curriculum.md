---
title: The Curriculum
description: Nine modules from Kubernetes basics to control-plane internals and real production war stories.
---

Kubelings is built as one continuous arc: start with a pod you can't keep alive,
finish by untangling multi-system cascade failures reproduced from real, cited
production postmortems. By the end you should know Kubernetes **as a platform,
in and out** — not just how to use it, but how it works and how it fails.

Every module is a set of small, broken-on-purpose clusters with automated checks.
Progress top to bottom; later modules assume the earlier ones.

## The nine modules

| # | Module | You learn | Status |
|---|--------|-----------|--------|
| 1 | **Foundations** | pods, Deployments, Services, namespaces, labels & selectors, the triage loop (`describe` → `logs` → fix → watch) | ✅ 7 lessons live |
| 2 | **Workloads** | rolling updates, blue/green & canary, DaemonSets, StatefulSets, Jobs, CronJobs, HPA, OOMKill & right-sizing, CPU throttling, probes, init containers, PDBs, QoS | ✅ 14 lessons live |
| 3 | **Config & Storage** | ConfigMaps, Secrets, PV/PVC lifecycle, StorageClasses, access modes, finalizer traps, kustomize | ✅ 6 lessons live |
| 4 | **Networking** | Services & endpoints, Ingress & Gateway API, NetworkPolicy, CoreDNS & the ndots amplifier, kube-proxy dataplane, conntrack, graceful shutdown | ✅ 9 lessons live |
| 5 | **Scheduling & Placement** | affinity/anti-affinity, taints & tolerations, topology spread, priority & preemption, noisy neighbors | ✅ 5 lessons live |
| 6 | **Security** | RBAC, ServiceAccounts & tokens, Pod Security, admission webhooks, container hardening, CIS benchmarks, egress lockdown, image digests, supply chain & runtime security | ✅ 10 lessons live |
| 7 | **Internals** | API server request & admission flow, watch/informers & APF, etcd (incl. backup/restore), CRDs & building operators, scheduler internals, controller reconciliation, kubelet ↔ CRI, leader election | ✅ 9 lessons live |
| 8 | **Observability & SRE** | events forensics, node NotReady triage, quotas, disk pressure & eviction, cluster upgrades, debugging playbooks | ✅ 7 lessons live |
| 9 | **War Stories (capstone)** | multi-concept cascade incidents from cited postmortems — everything at once, then the final boss | ✅ 5 lessons live |

## What finishing this actually gives you (and what it doesn't)

A finisher can honestly claim: **production troubleshooting reflexes** (the
describe → events → ownership-chain ladder, drilled ~60 times), a working
**internals model** (API flow, admission, watch/informers, etcd, scheduler,
controllers — including how you'd build one), **security posture** (RBAC →
tokens → admission → hardening → egress → supply chain, plus a real breach
walked end-to-end), **networking failure literacy** (endpoints, DNS,
conntrack, the kube-proxy dataplane), and pattern recognition from 20+ cited
real outages. Cert-wise: most of CKA/CKAD, and CKS-aware.

Deliberate non-goals — go elsewhere for these:

- **Executing host-level operations** — kubeadm upgrades, `etcdctl` restores,
  AppArmor/Falco installs are covered as full runbooks (readings), but lesson
  scripts never touch a host by design ([Security](/reference/security/)).
- **Helm execution & image building** — the toolchain isn't in the lesson
  shell; kustomize hands-on covers the declarative-config ground.
- **Service mesh, multi-cluster, cloud-provider specifics** (EKS/GKE/AKS
  quirks), and **Prometheus stack operation** — adjacent ecosystems, not
  Kubernetes fundamentals. The course tells you *when* you've reached their
  doorstep.

## Real incidents, woven in

Single-concept production incidents are reproduced as **runnable lessons inside
the module that teaches the concept** — you get the same broken state the
original team faced, on your own `kind` cluster, with a check that only passes
when you've fixed it. Multi-concept cascades land in Module 9.

Browse them all in the [Incident Library](/reference/incident-library/): every
`[REAL]` entry cites its public postmortem; synthetic composites are labeled
`[PATTERN]`.

## Where the checks run

Every lesson runs identically on [iximiuz Labs](https://labs.iximiuz.com/courses/kubelings-dbd840c8)
and locally on `kind` from one source of truth — see
[Getting Started](/getting-started/) and [Architecture](/reference/architecture/).
