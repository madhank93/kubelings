---
kind: unit
title: "etcd backup & restore: the runbook you rehearse before you need it"
name: etcd-backup-restore-unit
---


> **Reading.** The commands here run *on a control-plane host* as root against
> etcd's data and certs — outside this course's kubectl-only sandbox by
> design. Study the runbook now; rehearse it for real on a disposable cluster
> (kind is perfect for this — break one on your own machine). CKA expects
> these commands cold.

## Why this is the runbook that matters

Lesson 7.3 established it: **etcd is the cluster** — every object you've ever
applied is a key under `/registry/...`. Lose etcd without a backup and the
cluster isn't degraded, it's *gone*: every Deployment, Secret, CRD, and
namespace, unrecoverable. And when Reddit's Pi-Day upgrade (9.3) went
sideways, the path back was exactly this runbook — executed under pressure,
never rehearsed, five hours.

## Taking a snapshot

etcd speaks TLS to everyone, including its admin tool, so every command
carries three cert flags — memorize the shape, not the paths:

```sh
ETCDCTL_API=3 etcdctl snapshot save /backup/etcd-$(date +%F).db \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key
```

Verify it immediately — an unverified backup is a hope, not a backup:

```sh
etcdctl snapshot status /backup/etcd-2026-07-07.db --write-out=table
# hash | revision | total keys | size — nonzero keys or it's garbage
```

What's *in* it: a point-in-time copy of the whole keyspace. What's *not*:
anything on nodes (images, volumes — that's Module 3's storage story), and
nothing is encrypted unless you configured encryption at rest (6.8) — **an
etcd snapshot contains every Secret in the cluster; store it like one.**

## Restoring — the part with the sharp edges

The mental model that prevents most mistakes: **restore does not "load data
into" your etcd. It creates a brand-new data directory, and you point etcd at
it.**

```sh
# 1. Restore the snapshot into a NEW dir (never the live one)
ETCDCTL_API=3 etcdctl snapshot restore /backup/etcd-2026-07-07.db \
  --data-dir=/var/lib/etcd-restored

# 2. Stop the API server & etcd (kubeadm: move the static-pod manifests out)
mv /etc/kubernetes/manifests/kube-apiserver.yaml /root/
mv /etc/kubernetes/manifests/etcd.yaml /root/

# 3. Point etcd at the restored dir (edit etcd.yaml's hostPath volume:
#    /var/lib/etcd → /var/lib/etcd-restored), then put the manifests back
mv /root/etcd.yaml /root/kube-apiserver.yaml /etc/kubernetes/manifests/
```

The kubelet notices the static-pod manifests reappear (7.4's mechanism) and
brings the control plane back — now serving the snapshot's world.

The sharp edges, in the order they cut people:

- **Time travel is total.** Everything after the snapshot is gone — but the
  *data plane* is still running the newer reality. Controllers reconcile the
  restored spec against live state (7.1): pods created since the snapshot get
  orphaned or killed, pods deleted since come back. A restore is not "undo
  one mistake"; it's "move the whole cluster's brain back in time."
- **Multi-node etcd**: restore on *every* member with `--initial-cluster`
  flags matching the new cluster topology — a restored member believes it's
  founding a new cluster (new cluster ID), and mixing restored with
  non-restored members fails with cluster-ID mismatch. Quorum math from 7.3
  applies to the new cluster from second one.
- **The certs weren't in the snapshot.** PKI lives in
  `/etc/kubernetes/pki/`, on disk. Reddit's long tail was partly this
  category — the cluster's *identity* (certs, node bootstrap, addon config)
  is outside etcd, and your DR plan must cover both or restore day becomes
  discovery day.

## The actual prevention program

- **CronJob-shaped snapshots** (2.5) on control-plane nodes, shipped
  **off-cluster** (a backup stored in the cluster it backs up shares its
  fate — the Datadog lesson, M8, again), with `snapshot status` verification
  and retention.
- **Rehearse the restore quarterly** on a throwaway cluster: snapshot from
  prod, restore, `kubectl get all -A`, count objects. The runbook above
  should be a script with your paths in it before the day it's needed.
  Untested restores are recovery *hypotheses* (9.3).
- **Alert on backup age**, not just backup jobs succeeding — a CronJob that's
  been Suspended for a month passes every run it doesn't make.
- Managed clusters (EKS/GKE/AKS): the provider snapshots etcd — but *your*
  restore story is "recreate from git" (3.6's GitOps argument). Velero and
  friends cover the object-level backup niche in between.

*No check — study, then advance. And genuinely: go break a kind cluster and
restore it once. Twenty minutes now buys you five hours someday.*
