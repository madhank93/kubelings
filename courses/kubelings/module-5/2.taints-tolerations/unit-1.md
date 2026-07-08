---
kind: unit
title: "Everything Pending: who tainted the nodes?"
name: taints-tolerations-unit
---


## The situation

The morning standup opens with: *"nothing new deploys since last night — every
pod Pending, cluster-wide."* Existing pods run fine. Only *new* ones freeze.

```sh
kubectl -n kubelings describe pod -l app=invoices | grep -A4 -i events
```

```
0/3 nodes are available:
  1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: },
  2 node(s) had untolerated taint {dedicated: batch}.
```

Read the scheduler's rejection note carefully — it's a complete census. The
control-plane repelling pods is normal (that taint ships with Kubernetes, it's
why your workloads never land on the control plane). But `dedicated=batch` on
both workers? That's new.

**Taints** are the node saying *"keep off unless invited."* Three effects:

- `NoSchedule` — new pods won't place here (existing unaffected — which is why
  last night's pods still run!)
- `PreferNoSchedule` — soft version
- `NoExecute` — new pods won't place AND existing ones get **evicted**

**Tolerations** are the pod's invitation — matching key/value/effect lets a pod
ignore the taint. Taint = lock on the node; toleration = key in the pod.

Git blame the taint: last week's "dedicated batch node pool" project — cancelled
on Friday. The taints weren't.

## Your task

The migration is dead; its policy must die too:

1. See the taints for yourself (`kubectl describe node <worker> | grep -i taint`).
2. Remove `dedicated=batch:NoSchedule` from the workers — the *cancelled*
   project's leftovers. (Leave the control-plane taint alone.)
3. `invoices` schedules and goes Available on its own — no pod changes needed.

<details>
<summary>Hint</summary>

Trailing `-` removes a taint:

```sh
for n in $(kubectl get nodes -o name | grep -v control-plane); do
  kubectl taint node ${n#node/} dedicated=batch:NoSchedule-
done
```

Watch the Pending pods place themselves within seconds — the scheduler retries
continuously.

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


## Root cause

`dedicated=batch:NoSchedule` on every worker, from a cancelled project. With the
control-plane also (correctly) tainted, **zero** schedulable nodes remained for
un-tolerating pods. Classic properties of taint incidents:

- **Delayed fuse:** applied at night, nothing breaks — until the first new pod.
- **Asymmetric blast:** running workloads untouched (`NoSchedule` ≠ `NoExecute`),
  so dashboards stay green while deploys silently die.
- **Invisible in the app's namespace:** the cause lives on Nodes — cluster
  scope — while the symptom shows in every namespace.

## Fix

```sh
kubectl taint node <worker1> dedicated=batch:NoSchedule-
kubectl taint node <worker2> dedicated=batch:NoSchedule-
```

If the pool *had* shipped, the alternative is the toleration side:

```yaml
tolerations:
  - {key: dedicated, operator: Equal, value: batch, effect: NoSchedule}
```

…plus a `nodeSelector`/affinity to actually *prefer* the pool — a toleration
only unlocks the door, it doesn't walk through it. Dedicated pools need both.

## Taints you'll meet in the wild

| Taint | Who sets it | Meaning |
|---|---|---|
| `node-role.kubernetes.io/control-plane:NoSchedule` | kubeadm/kind | keep workloads off the control plane |
| `node.kubernetes.io/not-ready:NoExecute` | node controller | node unhealthy — pods evicted after `tolerationSeconds` (default 300) |
| `node.kubernetes.io/memory-pressure:NoSchedule` | kubelet | eviction manager active (Module 8) |
| `dedicated=<team>:NoSchedule` | humans | reserved pool — the pattern from this lesson |

The `not-ready` one matters: **every pod carries a hidden 300s toleration for
it** — that's the built-in delay between a node dying and its pods being
rescheduled. Now you know where those 5 minutes come from.

## Prevention

- Taints belong in IaC with the project that owns them — cancel the project,
  the taint dies in the same PR.
- Audit habit: `kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.taints}{"\n"}{end}'`
- Alert on cluster-wide Pending growth: it's this, resources, or PDBs — a
  three-item checklist you now fully own.

</details>
