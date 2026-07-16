## The situation

The scheduler has a mystique — "it places your pods" — that makes it feel
complicated and untouchable. Time to demystify it by doing its job by hand.

There's a Pending pod in `kubelings`:

```sh
kubectl -n kubelings get pod manual-sched
```

```
NAME           READY   STATUS    RESTARTS   AGE
manual-sched   0/1     Pending   0          1m
```

It's Pending not because of resources or taints, but because its
`schedulerName` points at a scheduler that doesn't exist — so the default
scheduler never touches it. It will sit here **forever** unless something writes
one field: `spec.nodeName`.

Because that's the scheduler's entire output. Not "running the pod" (that's the
kubelet). Not "creating the pod" (that's a controller). The scheduler is a loop
that watches for pods with an empty `nodeName`, decides which node fits, and
**writes the node's name into that field.** One string. The kubelet on the named
node then notices "a pod is assigned to me" and runs it.

## Your task

Be the scheduler:

1. Look at the cluster's nodes and pick a schedulable **worker** (the
   control-plane is tainted — Module 5).
2. Assign `manual-sched` to it by setting `spec.nodeName`.
3. Watch the kubelet run it — Pending → Running with no scheduler involved.

```sh
kubectl get nodes
kubectl -n kubelings get pod manual-sched -o jsonpath='{.spec.schedulerName}'
```
