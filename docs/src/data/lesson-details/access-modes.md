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
