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
