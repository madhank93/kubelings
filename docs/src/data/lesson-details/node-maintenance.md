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
