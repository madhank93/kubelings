---
kind: unit
title: "The reconcile loop: why deleted pods come back"
name: reconcile-loop-unit
---


## The situation

You've used Kubernetes for six modules. Now look under the hood at the one idea
holding it all up.

Delete a pod from a Deployment:

```sh
kubectl -n kubelings get pods -l app=resilient
kubectl -n kubelings delete pod <one-of-them>
kubectl -n kubelings get pods -l app=resilient -w    # watch — Ctrl-C when a new one is Running
```

Within a second, a replacement appears with a new name. Not resurrection — a
**controller** noticed reality (2 pods) no longer matched desire (3) and created
one. That gap-closing is **reconciliation**, and it is the entire operating
principle of Kubernetes:

```
for ever:
    desired = read spec from API server
    actual  = observe the world
    if desired != actual:
        take one step to close the gap
```

No central orchestrator scripts your cluster. Dozens of small controllers each
watch one kind of object and drive it toward its spec, independently, forever.
Deployment controller manages ReplicaSets; ReplicaSet controller manages Pods;
node controller manages node health; and so on. Declarative "desired state" works
*because* something is always reconciling toward it.

## Your task

Make the reconcile loop visibly fire:

1. Delete at least one `resilient` pod.
2. Watch the ReplicaSet controller recreate it (the deployment returns to 3/3).
3. The check confirms a replacement actually happened (≥ 4 `SuccessfulCreate`
   events: 3 initial + ≥ 1 heal).

```sh
kubectl -n kubelings delete pod -l app=resilient --field-selector=status.phase=Running --wait=false
kubectl -n kubelings get events --field-selector reason=SuccessfulCreate
```

<details>
<summary>Hint</summary>

Delete one and let it heal:

```sh
POD=$(kubectl -n kubelings get pods -l app=resilient -o name | head -1)
kubectl -n kubelings delete $POD
kubectl -n kubelings rollout status deploy/resilient
kubectl -n kubelings describe rs -l app=resilient | grep -A6 Events
```

You'll see `SuccessfulCreate` in the ReplicaSet's events — the controller
signing its work.

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


## Who did what (the chain of controllers)

```
you delete a Pod
  └─ API server records it gone, emits a watch event
      └─ ReplicaSet controller (watching pods with its selector) sees actual=2 < desired=3
          └─ creates 1 Pod with an ownerReference back to the ReplicaSet
              └─ scheduler assigns it a node
                  └─ kubelet on that node starts the container
```

Each arrow is an independent component reacting to a watch event. Nobody
"orchestrated" this — it's emergent from many loops sharing one source of truth
(etcd, via the API server).

## ownerReferences: the thread that makes it work

```sh
kubectl -n kubelings get pod -l app=resilient -o jsonpath='{.items[0].metadata.ownerReferences}'
```

Each pod points at its ReplicaSet; each ReplicaSet at its Deployment. That's how
the controller knows *which* pods are "its" to count — and how `kubectl delete
deployment` cascades (garbage collection walks ownerReferences down). Break the
labels/selector and the controller adopts or orphans pods accordingly — the
Module 1 selector lesson, seen from the controller's side.

## Why this reframes everything

- "It won't stay deleted" → a controller is reconciling; delete the *owner*
  (Deployment), not the pod.
- "It won't stay changed" (edit a managed pod, it reverts) → same reason; edit
  the spec (template), not the child.
- Level-triggered, not edge-triggered: controllers act on *observed state*, not
  on the event that got them there — so they self-heal even after missing
  events, restarts, or partitions. Robustness by design.

## Where next

Every remaining internals lesson is a variation on this loop: the scheduler
reconciles unscheduled→scheduled, the endpoints controller reconciles
service→endpoints, leader election decides *which replica* of a controller runs
the loop. Hold onto this one diagram; the rest is detail.

</details>
