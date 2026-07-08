---
kind: unit
title: "The upgrade runbook: version skew, kubeadm, and not becoming a war story"
name: upgrade-runbook-unit
---


> **Reading.** Upgrades run `kubeadm` and package managers on hosts — outside
> the kubectl sandbox. But you've already done every *cluster-side* move this
> runbook uses (drain, PDBs, cordon, uncordon), and you've read what happens
> when upgrades go wrong (Reddit, 9.3). This assembles the pieces into the
> procedure.

## The contract that makes it possible

Kubernetes upgrades **rolling, one minor version at a time**, because of the
documented **version skew policy** — who may be older than whom:

```
kube-apiserver     vN         (upgrade FIRST — nothing may be newer than it)
controller-mgr,    vN or vN-1
scheduler
kubelet            vN … vN-3  (nodes may lag up to three minors)
kubectl            vN±1
```

Two operational consequences:

- **Control plane first, always.** A kubelet newer than its API server is
  unsupported territory. And skew rules bind *within* a node too: CircleCI's
  [2023-03-14 outage](https://discuss.circleci.com/t/incident-report-2023-03-14-delays-starting-jobs/47555)
  came from kubelet and kube-proxy at incompatible versions mid-upgrade —
  the iptables ruleset format changed between them, every sync corrupted the
  node's rules a little more, and recovery was a node-by-node restart.
- **Never skip minors.** 1.28 → 1.30 means 1.28 → 1.29 → 1.30, control plane
  then nodes, each step. (The kubelet's N-3 allowance is what lets huge
  fleets upgrade nodes on a slower cadence than control planes.)

## The kubeadm sequence

**Control plane node:**

```sh
kubeadm upgrade plan                  # what's available, what it will touch
# upgrade kubeadm itself (apt/yum), then:
kubeadm upgrade apply v1.31.x         # first CP node
kubeadm upgrade node                  # remaining CP nodes
# then per CP node: upgrade kubelet+kubectl packages, restart kubelet
```

`upgrade apply` rewrites the static-pod manifests (7.4) — API server,
controller-manager, scheduler restart on new versions one by one; with
multiple control-plane nodes and leader election (7.4), the cluster never
loses its brain. etcd gets upgraded by kubeadm too — **which is why the
snapshot from the previous lesson is step zero.**

**Each worker, one (or a few) at a time — every verb is a lesson you've done:**

```sh
kubectl drain <node> --ignore-daemonsets --delete-emptydir-data   # 2.11
# on the node: upgrade kubeadm pkg → kubeadm upgrade node
#              → upgrade kubelet pkg → systemctl restart kubelet
kubectl uncordon <node>                                           # 8.4 —
kubectl get nodes                                                 # ...and don't forget it
```

Drain respects PDBs (2.11) — this is *the* moment mis-sized PDBs block fleet
maintenance, and the taint/cordon leftovers lesson (8.4) is literally "someone
ran this runbook and skipped the last step." Watch workloads reschedule
(M5's spreading rules decide where) before draining the next node.

## What breaks anyway (the pre-flight checklist)

Every item below is a real incident class — most from this course:

1. **API deprecations**: removed APIs (`kubectl api-resources`, deprecation
   warnings on apply) — manifests using them fail *after* the upgrade. Scan
   rendered manifests (3.6) against the target version before touching prod.
2. **Platform label/behavior renames**: Reddit's `master` →`control-plane`
   label rename killed their CNI (9.3). Grep addon configs — CNI, ingress,
   monitoring — for anything selecting on platform-owned labels or flags.
3. **CNI/CSI/addon compatibility**: the CNI has its own support matrix
   against Kubernetes minors; upgrade order is CNI-doc-driven, not guessed.
   (The layer below Kubernetes — Datadog's lesson, 8.6.)
4. **PDB math**: `kubectl get pdb -A` — any `ALLOWED DISRUPTIONS: 0` will
   hang the first drain (2.11). Fix them *before* the window.
5. **Snapshot etcd** (7.6) and know your restore works. Downgrades are
   unsupported; the snapshot is the only reverse gear.
6. **Canary a node**: upgrade one worker, soak it with real traffic for a
   bake period (the OpenAI lesson, 9.2: longer than your caches), then wave
   through the rest.

## Managed clusters

EKS/GKE/AKS move the control-plane upgrade behind a button/API — the skew
policy, node drains, PDB math, deprecation scanning, and addon compatibility
remain entirely yours. The runbook shrinks by one section; the checklist
doesn't shrink at all.

## The meta-lesson

Upgrades are the highest-leverage *scheduled* risk in cluster operations:
three of this course's nine war-story incidents (Reddit, Datadog, and the
drain that never finishes) are upgrade-adjacent. The pattern that survives
contact: **small blast radius (one node, one minor), verified state between
steps (`get nodes`, `get pdb`, workload health), and a rehearsed reverse
gear.** The teams that upgrade quarterly do it calmly; the teams that
upgrade every three years star in Module 9.

*No check — study, then advance.*
