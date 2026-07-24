---
kind: unit
title: "etcd NOSPACE: compact, defrag, disarm"
name: etcd-defrag-unit
---


> **☁ iximiuz Labs only.** You'll drive `etcdctl` against a real etcd's live
> backend — compaction, defragmentation, alarm management — on a control plane
> you own. That's root, the etcd certs, and a database you're allowed to push
> into read-only. Not a kubectl-sandbox operation.

## The failure that quietly stops every write

7.6 was losing etcd. This is a subtler etcd emergency: etcd is *up*, healthy,
serving reads — and the entire cluster has silently gone **read-only**. New
pods won't schedule, `kubectl apply` hangs or errors, controllers can't update
status. And `kubectl get` works fine, which is what makes it so confusing.

The cause is etcd's **backend quota**. etcd caps the size of its on-disk
database (`--quota-backend-bytes`, default ~2 GiB). Cross it and etcd raises a
cluster-wide **`NOSPACE`** alarm — `mvcc: database space exceeded` — and
refuses all writes until you clear it. It's a guardrail: a full backend that
kept accepting writes would corrupt, so etcd stops first.

## Why the backend fills: MVCC history

etcd is multi-version. Every write creates a new **revision**; old revisions
aren't deleted, they accumulate. A busy cluster — lots of updates to the same
objects (leases, endpoints, node status) — piles up revision history fast. The
*logical* keyspace is small, but the *physical* backend is large because it's
holding every past version.

That gives you the two-step fix, and it's important they're different
operations:

- **Compaction** discards revision history older than a point — it frees space
  *inside* etcd's logical view, but the file on disk doesn't shrink.
- **Defragmentation** actually rewrites the backend file and returns the freed
  pages to the filesystem. This is what shrinks `dbSize`.

Compact without defrag and the disk is still full. Defrag without compact and
there's little to reclaim. You do both.

## The NOSPACE runbook

Everything runs through `etcdctl`, and on this distroless etcd there's no shell
in the pod — so you `kubectl exec` `etcdctl` itself, one command at a time,
with the three cert flags (7.6's shape):

```sh
# is it armed, and how big is the backend?
etcdctl alarm list                       # "memberID:... alarm:NOSPACE"
etcdctl endpoint status -w table         # DB SIZE column

# 1 · compact away old revisions (use the current revision)
REV=$(etcdctl endpoint status -w json | grep -o '"revision":[0-9]*' | head -1 | cut -d: -f2)
etcdctl compact "$REV"

# 2 · defragment — rewrites the file, returns space to disk (can take a while)
etcdctl defrag --command-timeout=60s

# 3 · disarm the alarm — LAST
etcdctl alarm disarm
```

**Order matters, and disarm is last.** The alarm is a latch: disarming it while
the backend is still over quota just re-arms on the very next write. You reclaim
space *first*, confirm `dbSize` dropped, and only then disarm.

And if the backend is legitimately large — the quota is genuinely too small for
the cluster — reclaiming isn't enough. You raise `--quota-backend-bytes` in
`/etc/kubernetes/manifests/etcd.yaml` (a static-pod edit, 7.4) to a sane value
before you disarm. Defrag buys you room; right-sizing keeps you out of the
alarm.

## Your turn

`init` set etcd's quota below its current size, so the API server's own writes
tripped the alarm. The cluster is read-only right now.

Clear it, on **cplane-01**:

1. Confirm the alarm and read the backend size — `etcdctl alarm list`,
   `etcdctl endpoint status -w table`.
2. **Compact** to the current revision, then **defrag** to reclaim disk.
3. The quota `init` set is too small to stay under — raise
   `--quota-backend-bytes` in `/etc/kubernetes/manifests/etcd.yaml` back to a
   sane value (the kubeadm default is fine) and let etcd restart.
4. **Disarm** the alarm.

The check writes a probe object and confirms it sticks *without* re-arming
NOSPACE — so a bare `disarm` that leaves the quota too small won't pass.

<details>
<summary>Hint</summary>

`etcdctl` lives inside the etcd pod (the host has no copy, and the image is
distroless — no shell, so run one command per `kubectl exec`):

```sh
E() { kubectl -n kube-system exec etcd-cplane-01 -c etcd -- etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key "$@"; }

E alarm list
E endpoint status -w table
```

Reclaim space, then right-size, then disarm — in that order. Because `init`
made the quota smaller than the database, compact+defrag alone can't get you
under it; you must raise `--quota-backend-bytes` in the etcd manifest
(`/root/kubelings-etcd-baseline.yaml` has the original) and let the kubelet
rebuild the pod before the final `E alarm disarm`. Disarming while still over
quota re-arms on the next write.

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

```sh
E() { kubectl -n kube-system exec etcd-cplane-01 -c etcd -- etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key "$@"; }

# 1 · see it
E alarm list                       # NOSPACE
E endpoint status -w table         # DB SIZE at/over quota

# 2 · reclaim: compact old revisions, then defrag to shrink the file
REV=$(E endpoint status -w json | grep -o '"revision":[0-9]*' | head -1 | cut -d: -f2)
E compact "$REV"
E defrag --command-timeout=60s

# 3 · right-size the quota (init set it below the DB size on purpose).
#     Restore the kubeadm-default manifest, or raise the value in place.
cp /root/kubelings-etcd-baseline.yaml /etc/kubernetes/manifests/etcd.yaml
until E endpoint health >/dev/null 2>&1; do sleep 2; done   # etcd pod rebuilt

# 4 · disarm LAST — now that we're under quota it stays cleared
E alarm disarm
E alarm list                       # empty

# 5 · prove writes work again
kubectl -n kube-system create configmap defrag-ok --from-literal=ok=1
```

The baseline manifest carries the kubeadm default quota (~2 GiB), which is why
restoring it is the simplest "right-size." In production you'd tune the value
to the cluster, not just revert.

</details>

## Root cause, restated

NOSPACE is etcd protecting itself: a backend that filled up stops taking writes
before it can corrupt. The cluster looks half-alive — reads fine, writes dead —
and the fix is a specific, ordered runbook.

- **Reads work, writes don't = suspect etcd's alarm.** `etcdctl alarm list` is
  the first thing to check when `kubectl get` is fine but `apply` hangs.
- **Compact and defrag are different jobs.** Compaction drops MVCC history;
  defragmentation rewrites the file to actually return space to disk. You need
  both, and defrag is what shrinks `dbSize`.
- **Disarm last, and right-size if you must.** The alarm re-arms the instant a
  write lands while you're still over quota. Reclaim space and, if the quota is
  genuinely too small, raise `--quota-backend-bytes` before disarming.
