---
kind: unit
title: "One disk, two nodes: the access-mode trap"
name: access-modes-unit
---


## The situation

Last sprint's retro action item: *"wiki is a single point of failure — scale it
to 2 and spread the replicas."* Done and done: `replicas: 2`, required
anti-affinity across hostnames. Ship it.

```
NAME                    READY   STATUS    RESTARTS   AGE
wiki-6f8d7c9b5d-8mkwp   1/1     Running   0          3d
wiki-6f8d7c9b5d-tq2xn   0/1     Pending   0          3d
```

Three days Pending. The scheduler's explanation:

```sh
kubectl -n kubelings describe pod -l app=wiki | grep -A4 -i events
```

```
0/3 nodes are available: 1 node(s) had volume node affinity conflict,
1 node(s) didn't match pod anti-affinity rules, ...
```

Unpack the deadlock:

- The shared PVC is **ReadWriteOnce** — mountable read-write by **one node** at
  a time (not one pod — one *node*; two pods on the same node can share it).
- This cluster's `standard` class is node-local storage: once bound, the volume
  physically lives on the node where the first pod landed.
- Anti-affinity *requires* pod 2 on a **different** node… where the volume can
  never be.

Pod 2 must be where the disk isn't. Checkmate.

## Your task

Make `wiki` fully Available (no Pending pods). Think about what "HA" honestly
means for a stateful app on RWO storage, then pick a consistent design:

- drop the hard spread requirement so both replicas can share the volume's node, or
- one replica per volume (that's StatefulSet + `volumeClaimTemplates` — Module
  2's statefulset lesson), or
- for this lesson's scope: any solution with zero Pending pods and the data
  still mounted.

```sh
kubectl -n kubelings describe pvc wiki-data | grep -i 'node\|events' -A2
kubectl -n kubelings get pods -l app=wiki -o wide
```

<details>
<summary>Hint</summary>

Smallest honest fix — remove the anti-affinity so the scheduler may co-locate
with the volume (volume topology then pulls pod 2 to the same node):

```sh
kubectl -n kubelings patch deploy wiki --type=json -p '[
  {"op":"remove","path":"/spec/template/spec/affinity"}
]'
```

Both replicas on one node isn't node-HA — that's the lesson. Real HA for this
app = StatefulSet with a volume per replica, or RWX storage.

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


## Access modes decoded

| Mode | Promise | Typical backing |
|---|---|---|
| **RWO** ReadWriteOnce | one **node** read-write | block disks (EBS, PD, local-path) |
| **ROX** ReadOnlyMany | many nodes, read-only | prebaked content |
| **RWX** ReadWriteMany | many nodes read-write | NFS, CephFS, EFS/Filestore |
| RWOP ReadWriteOncePod | exactly one **pod** | CSI, when even node-sharing is unsafe |

The scaling PR assumed replicas share data like they share a Service. Storage
doesn't work that way: RWO is a *node-attachment* contract, and most default
StorageClasses are RWO block storage.

## Fix (lesson scope)

```sh
kubectl -n kubelings patch deploy wiki --type=json -p '[
  {"op":"remove","path":"/spec/template/spec/affinity"}
]'
kubectl -n kubelings rollout status deploy/wiki
```

Scheduler + volume topology co-locate both replicas with the disk. Available:
yes. Node-HA: no — and now that trade-off is *visible* instead of hidden behind
a Pending pod.

## The real designs

- **State per replica** → StatefulSet + `volumeClaimTemplates` (each pod its own
  RWO volume) and app-level replication.
- **Genuinely shared files** → RWX-capable storage (NFS/CephFS/EFS) — and accept
  its performance/locking semantics.
- **Multiple writers to one file tree over RWX** is where corruption stories
  come from; most apps that "just need shared disk" actually need a database.

## Prevention

- Before any `replicas: 1 → N` on a pod with a PVC: check the claim's access
  mode first. It's one field and it predicts this entire incident.
- `volume node affinity conflict` in scheduler events = storage pinned somewhere
  your placement rules forbid. The fix is a design decision, not a retry.

</details>
