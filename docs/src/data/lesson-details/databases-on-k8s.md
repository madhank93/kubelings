> **Reading.** No lab: the failure this teaches is *silent data loss during
> failover*, and the honest way to experience it is on a cluster you're allowed
> to lose. What follows is the decision framework — and the specific questions to
> ask before someone puts your primary database behind a StatefulSet.
>
> Sources:
> [Gravitational (Teleport) — Experiences running PostgreSQL on Kubernetes](https://goteleport.com/blog/running-postgresql-on-kubernetes/) ·
> [Kubernetes docs — StatefulSet limitations](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#limitations)

## The claim to be suspicious of

"StatefulSets give you stateful workloads." They give you three things, and they
are all *naming and ordering*:

1. **Stable identity** — `db-0`, `db-1`, and per-pod DNS that survives restarts.
2. **Stable storage** — each ordinal keeps its own PVC across rescheduling.
3. **Ordered, controlled rollout** — one pod at a time, in order, on request.

That's the whole contract. Everything a database actually needs — *who is the
primary, is the replica caught up, is it safe to promote, is this volume's data
consistent* — Kubernetes does not know and cannot check.

## The failure that matters: promotion with lag

Gravitational's warning, in one sentence: **Kubernetes is not aware of the
deployment details of Postgres, and a naive deployment can lose data.**

The mechanism:

```
primary (db-0) ── async streaming replication ──▶ replica (db-1)
   commits at LSN 1000                              received to LSN 940
   node dies ───────────────────────────────────────────────────────▶
   something promotes db-1 ⇒ 60 transactions are gone, permanently
```

Nothing here is broken from Kubernetes' point of view. A liveness probe failed, a
pod was replaced, a Service selector moved. The 60 committed transactions were
acknowledged to clients that will never hear otherwise. **The orchestrator's
"self-healing" is the data-loss event.** And if the old primary comes back and
also accepts writes, you now have split-brain to reconcile by hand.

## What the storage layer does and doesn't promise

| You have | It guarantees | It does not guarantee |
|---|---|---|
| PVC + `ReadWriteOnce` | the volume re-attaches to the pod that owns it | that the data on it is *consistent* — that's fsync, WAL and the DB's job |
| Network storage (EBS/Ceph/NFS) | availability across node failure | latency you'd choose for a database, or protection against replicating corruption |
| Volume snapshots | a point-in-time copy of blocks | a *restorable database* unless the snapshot is crash-consistent or the DB was quiesced |
| Local PV (fast!) | node-local NVMe speed | any survival at all if that node dies — the pod is now pinned to a dead machine |

Filesystem/block replication has no domain knowledge: it will faithfully
replicate a corrupt page. Database replication (WAL shipping, streaming) does
have that knowledge — which is why *the database's own replication is the
durability mechanism*, and the volume is just where the bytes live.

## Where the ecosystem landed

Gravitational's 2018 conclusion — run databases on VMs with DBAs unless you have
real expertise — was correct for 2018 and is now the *default*, not the ceiling.
The gap it identified is exactly what **operators** were built to close:

| Question Kubernetes can't answer | Who answers it now |
|---|---|
| Who is the primary? | operator + a consensus/leader mechanism (Patroni's DCS, or the operator's own leader election on the API server) |
| Is the replica caught up enough to promote? | operator checks replication lag against a policy; **synchronous** replication makes the answer trivially yes |
| Is this failover or a network partition? | fencing — the old primary is demoted/killed before promotion |
| Can I restore this to 14:03 yesterday? | operator-managed continuous WAL archiving + PITR, tested by a restore job |

CloudNativePG, Zalando's postgres-operator, Percona/EDB, StackGres for Postgres;
Vitess, Strimzi, CockroachDB and friends elsewhere. The rule the incident
teaches survives all of them: **if nothing in your stack knows the replication
lag, nothing in your stack should be promoting anything.**

## Concept checks

- Your Postgres StatefulSet has 3 replicas and a Service pointing at `db-0`.
  `db-0`'s node dies. What actually happens, minute by minute? (Pod stays
  Pending/Terminating while the node is unreachable — ~5 min of taint tolerations
  — the PVC won't detach until the old attachment is released, and *nothing*
  promotes a replica because no controller in that picture knows how. Write
  outage, not failover.)
- Why is `ReadWriteOnce` a feature, not a limitation, for a database? (Two
  Postgres processes on one data directory corrupt it. The access mode is a
  guardrail — see M3 `access-modes` — and multi-attach is precisely what you
  don't want here.)
- What makes "we take nightly volume snapshots" a weaker claim than "we archive
  WAL continuously"? (RPO. A snapshot's worst case is 24 hours of loss and its
  restore may be crash-consistent at best; WAL archiving gives point-in-time
  recovery to the second — and both are only real if you've *restored* from
  them.)

## Takeaways

- **Decide with an RPO/RTO number, not a vibe.** "Zero data loss" means
  synchronous replication and the write-latency bill that comes with it. Anything
  async has a non-zero RPO — write it down and get it signed off.
- **Never let a probe be a promotion trigger.** Liveness restarts processes;
  promotion is a durability decision that needs lag + fencing.
- **If you run a database on Kubernetes, run an operator** — or don't run the
  database on Kubernetes. The middle option (hand-rolled StatefulSet + a
  failover script) is the one that produces postmortems.
- **PVC lifecycle is part of the design**, not an afterthought: reclaim policy,
  `persistentVolumeClaimRetentionPolicy` on the StatefulSet, and the finalizer
  traps from M3 `pattern-pvc-terminating`.
- **Practise the restore.** The same conclusion Reddit reached the hard way
  (M9 `incident-reddit-piday`): an untested restore is a hypothesis.

*No check — study, then advance.*

<figure class="lesson-diagram">
<img src="/diagrams/databases-on-k8s-failover.svg" alt="databases-on-k8s failover diagram" loading="lazy">
</figure>

