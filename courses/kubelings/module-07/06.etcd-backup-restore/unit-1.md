---
kind: unit
title: "etcd backup & restore: the runbook you rehearse before you need it"
name: etcd-backup-restore-unit
---


> **☁ iximiuz Labs only.** The commands here run *on a control-plane host* as
> root against etcd's data and certs — outside this course's kubectl-only
> sandbox by design, so this one can't run on your local `kind` cluster. Here
> you get a real, disposable control plane and you will actually destroy it:
> read the runbook, then do the drill at the bottom. CKA expects these
> commands cold.

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
# 1. Restore the snapshot into a NEW dir (never the live one). Recent etcd
#    (3.6+) moved restore into etcdutl; carry the node's identity and a fresh
#    cluster token so the restored data founds a new cluster it can't confuse
#    with the old one.
etcdutl snapshot restore /backup/etcd-2026-07-07.db \
  --name cplane-01 \
  --initial-cluster cplane-01=https://<node-ip>:2380 \
  --initial-advertise-peer-urls https://<node-ip>:2380 \
  --initial-cluster-token restored-$(date +%s) \
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

## Your turn

Twenty minutes now buys you five hours someday. Do it here, on a control
plane you're allowed to destroy.

`init` created **`kubelings/treasure`** — pretend it's the only copy of
something that matters — and noted its UID and etcd's current cluster ID.

Run the whole drill, in this order:

1. **Snapshot** etcd to `/backup/` while everything is still healthy, and
   verify the snapshot with `snapshot status`.
2. **Cause the disaster:** `kubectl -n kubelings delete configmap treasure`
3. **Restore** from your snapshot and bring the control plane back.
4. Confirm `treasure` is there again.

The order is load-bearing: snapshot *before* you break things, or you'll
restore a world that never had the ConfigMap in it.

The check verifies two independent things — that etcd is running a restored
cluster, and that `treasure` came back as the *same object*, not a
lookalike you typed in again.

<details>
<summary>Hint</summary>

The mental model that prevents most mistakes: **restore does not load data
into your running etcd. It writes a brand-new data directory, and you point
etcd at it.** So the sequence is: restore to a new dir → stop the control
plane → repoint `etcd.yaml`'s hostPath at that dir → let it come back.

Stopping and starting the control plane on kubeadm is done by moving static
pod manifests out of `/etc/kubernetes/manifests/` and back (7.4's mechanism)
— the kubelet is watching that directory.

Don't shortcut step 4 by running `kubectl create configmap treasure` again.
The check compares UIDs precisely because that shortcut is the thing people
reach for under pressure, and it isn't a restore.

If `kubectl` hangs while the manifests are moved out, that's correct — you
have deliberately taken the API server down. It comes back when they do.

</details>

::simple-task
---
:tasks: tasks
:name: verify_done
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>


The etcd image here is distroless and there's no `etcdctl`/`etcdutl` on the
host, so **save** runs inside the live etcd pod (the binary is in the image),
and **restore** — which happens while etcd is down — runs in a throwaway
container built from the same image, bind-mounting the data dir.

```sh
CERTS="--endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key"

# 1 · snapshot while healthy — save lands on the host via etcd's hostPath (/var/lib/etcd)
kubectl -n kube-system exec etcd-cplane-01 -c etcd -- \
  etcdctl $CERTS snapshot save /var/lib/etcd/snap.db
cp /var/lib/etcd/snap.db /backup/snap.db      # keep it off the data dir too

# 2 · the disaster
kubectl -n kubelings delete configmap treasure

# 3 · stop the control plane (kubeadm: move the static-pod manifests out)
mkdir -p /root/m
mv /etc/kubernetes/manifests/*.yaml /root/m/
sleep 12

# 4 · restore into a NEW data dir. etcd 3.6 moved restore into etcdutl, and
#     it must carry this node's identity — plus a FRESH cluster token, so the
#     restored data founds a genuinely new cluster (new cluster ID) that can't
#     accidentally rejoin the old one. PEER is this node's peer URL.
PEER=https://172.16.0.2:2380
ctr -n k8s.io run --rm \
  --mount type=bind,src=/var/lib,dst=/var/lib,options=rbind:rw \
  registry.k8s.io/etcd:3.6.8-0 etcd-restore \
  etcdutl snapshot restore /var/lib/etcd/snap.db \
    --name cplane-01 \
    --initial-cluster cplane-01=$PEER \
    --initial-advertise-peer-urls $PEER \
    --initial-cluster-token restored-$(date +%s) \
    --data-dir /var/lib/etcd-restored

# 5 · point etcd at the restored dir (edit the hostPath volume) and bring it back
sed -i 's#path: /var/lib/etcd$#path: /var/lib/etcd-restored#' /root/m/etcd.yaml
mv /root/m/*.yaml /etc/kubernetes/manifests/
until kubectl get --raw=/readyz >/dev/null 2>&1; do sleep 2; done

# 6 · it's back, a NEW cluster, and treasure is the SAME object it always was
kubectl -n kube-system exec etcd-cplane-01 -c etcd -- etcdctl $CERTS \
  endpoint status -w table            # cluster ID has changed
kubectl -n kubelings get configmap treasure -o jsonpath='{.metadata.uid}{"\n"}'
```

(On a stock kubeadm cluster `etcdctl` is often on the host — then save/status
run directly. Here it isn't, which is why save goes through the pod and
restore through a container. The moves are the same either way.)

## Root cause, restated

There's no failure to diagnose here — the failure is organizational, and it
happens before the incident. Every cluster has a backup story; most have an
untested one, which is a recovery *hypothesis*, not a recovery plan.

Three things this drill teaches that reading it doesn't:

- **Restore is a new cluster.** The cluster ID changes, which is why
  multi-node restores need matching `--initial-cluster` flags on every
  member, and why mixing restored and non-restored members fails outright.
- **Time travel is total.** Everything after the snapshot is gone, but the
  data plane is still running the newer reality. Controllers then reconcile
  the restored spec against live state: pods created since the snapshot get
  orphaned, pods deleted since come back. A restore is not "undo one
  mistake" — it moves the whole cluster's brain back in time.
- **The certs weren't in the snapshot.** PKI lives in `/etc/kubernetes/pki/`,
  on disk. The cluster's *identity* is outside etcd, and a DR plan that
  covers only etcd makes restore day into discovery day. That was a real
  part of Reddit's long tail.

And the thing to actually take away: an etcd snapshot contains every Secret
in the cluster in plaintext unless you did M6.16. Store it like the
credential dump it is.

</details>
