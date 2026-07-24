---
kind: unit
title: "What the scheduler actually does (bypass it to find out)"
name: scheduler-nodename-unit
---


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

<details>
<summary>Hint</summary>

`nodeName` is immutable once set, so patch this fresh-Pending pod (it has no node
yet):

```sh
WORKER=$(kubectl get nodes -o name | grep -v control-plane | head -1)
WORKER=${WORKER#node/}
kubectl -n kubelings patch pod manual-sched --type=merge -p "{\"spec\":{\"nodeName\":\"$WORKER\"}}"
kubectl -n kubelings get pod manual-sched -w
```

(If patch is rejected because the pod already has a node, delete and recreate it
with `nodeName` set in the manifest.)

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


## The scheduler's real loop (now that it's not magic)

For each pod with empty `nodeName`, two phases:

1. **Filter (predicates)** — eliminate nodes that *can't* work: not enough
   free requests, unsatisfied nodeSelector/affinity, untolerated taints, volume
   topology conflicts. Survivors are "feasible."
2. **Score (priorities)** — rank the feasible nodes: spread constraints, image
   locality, least/most allocated, affinity preferences. Highest score wins.
3. **Bind** — write `nodeName`. Exactly what you just did by hand.

Every scheduling behavior you learned in Module 5 is a filter or a score plugin:
taints/tolerations and required affinity are **filters**; topology spread and
preferred affinity are **scores**. You weren't learning features — you were
tuning this pipeline.

## What your bypass proved

- **`nodeName` set ⇒ scheduler skipped.** Set it in a manifest and the pod goes
  straight to the kubelet — how static pods and some operators pin placement.
  Also how you'd wedge a pod onto a full/tainted node (the scheduler's filters
  never run, so its protections don't apply — a footgun).
- **Pending with a healthy cluster** almost always means "no node passed the
  filters." `kubectl describe pod` prints the per-node reasons — that census you
  read in the taints and access-modes lessons is the filter phase talking.

## Custom schedulers & why schedulerName exists

`schedulerName` lets multiple schedulers coexist — run your own for a subset of
pods (batch/gang scheduling, GPU bin-packing) alongside the default. The Pending
pod here used a nonexistent name to *deny* it a scheduler; in production it's how
you *choose* one.

## Prevention / habits

- Reach for `nodeName` only knowingly — it skips every safety filter.
- "Pod stuck Pending" → `kubectl describe pod` → read the filter reasons; don't
  guess. The scheduler always tells you why it couldn't place something.

</details>
