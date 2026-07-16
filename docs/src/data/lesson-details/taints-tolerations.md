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
