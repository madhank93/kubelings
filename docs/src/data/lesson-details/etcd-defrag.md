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
