---
kind: unit
title: "Node maintenance: drain like you mean it"
name: node-maintenance-unit
---


## The situation

A kernel patch is due on one worker. Which one:

```sh
kubectl -n kubelings get configmap maintenance-target -o jsonpath='{.data.node}'
kubectl -n kubelings get pods -o wide
```

The target runs two tenants: the `node-agent` DaemonSet pod and the
`orders-batch` pod with scratch data on an `emptyDir`. In M2.11
(`pdb-blocks-drain`) a drain *hanging* was the problem to fix. This time
the drain is the job — done in the right order, with eyes open at each
refusal.

## Your task

Read the target node into a variable first:

```sh
NODE=$(kubectl -n kubelings get cm maintenance-target -o jsonpath='{.data.node}')
```

1. **Cordon** — stop new arrivals before evicting current tenants:

   ```sh
   kubectl cordon $NODE
   kubectl get node $NODE     # Ready,SchedulingDisabled
   ```

   Cordon only flips `spec.unschedulable` — nothing is evicted. Running a
   bare cordon during an incident is a scalpel: the node keeps serving,
   the scheduler just stops adding to it.

2. **Drain — and read both refusals before flagging past them:**

   ```sh
   kubectl drain $NODE
   # error: cannot delete DaemonSet-managed Pods … node-agent-…
   # error: cannot delete Pods with local storage … orders-batch-…
   ```

   - The DaemonSet pod *can't* be rehomed — its whole job is "one per
     node", and its controller would recreate it immediately.
     `--ignore-daemonsets` means *leave it, I understand*.
   - The `emptyDir` dies with the pod — Kubernetes refuses to silently
     destroy data. `--delete-emptydir-data` is you signing for the loss.

   ```sh
   kubectl drain $NODE --ignore-daemonsets --delete-emptydir-data
   kubectl -n kubelings get pods -o wide    # orders-batch reborn elsewhere
   ```

3. **Maintain** — the pretend patch. In this drill: label the node as the
   maintenance record:

   ```sh
   kubectl label node $NODE kubelings.dev/maintenance-done=true
   ```

4. **Uncordon** — reopen for business:

   ```sh
   kubectl uncordon $NODE
   ```

<details>
<summary>Hint</summary>

`orders-batch` won't move back after uncordon — evicted pods are
*replaced*, not remembered; the replacement stays where the scheduler put
it. That asymmetry is normal. Rebalancing (if you cared) is a
`rollout restart` away — or a descheduler's job in real fleets.

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


## Fix

```sh
NODE=$(kubectl -n kubelings get cm maintenance-target -o jsonpath='{.data.node}')
kubectl cordon $NODE
kubectl drain $NODE --ignore-daemonsets --delete-emptydir-data
kubectl label node $NODE kubelings.dev/maintenance-done=true
kubectl uncordon $NODE
```

## Why the flags exist (and what they're really asking)

Drain is deliberately obstinate — each refusal is a data-loss or
availability question aimed at you:

| Refusal | The question | Your answer |
|---|---|---|
| DaemonSet pods | "this can't move — proceed around it?" | `--ignore-daemonsets` (agent tolerates the taint and keeps running through maintenance — usually exactly right for log/metric shippers) |
| emptyDir pods | "this data dies — accept?" | `--delete-emptydir-data` (scratch: yes; anything that hurts to lose had no business in emptyDir — M3's storage lessons) |
| PDB at its limit | "this eviction breaks an availability promise — wait?" | drain *waits*, and M2.11 is the story of waiting forever |

Under the hood drain uses the **eviction API** (respects PDBs), not bare
deletes — that's why a PDB can stall it and why it's the polite tool.

## Prevention / takeaway

- The cycle is always cordon → drain → work → uncordon. Cordon-first means
  no pod gets scheduled onto a node that's about to go down mid-incident.
- Evicted pods don't come home after uncordon; expect the post-maintenance
  imbalance and let it be (or restart rollouts deliberately).
- This exact cycle, node by node, is the inner loop of the cluster upgrade
  runbook (M8.7) — there it's automated by kubeadm/managed pools, but the
  refusals and flags are the same ones.
- Real fleets record maintenance in systems, not labels — but "leave a
  queryable trail" (`kubectl get nodes -l kubelings.dev/maintenance-done`)
  is the habit this drill plants.

</details>
