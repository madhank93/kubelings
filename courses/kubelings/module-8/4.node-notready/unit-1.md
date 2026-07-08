---
kind: unit
title: "The morning after NotReady"
name: node-notready-unit
---


## The situation

The 3:12 a.m. page: a worker node went **NotReady** — its kubelet TLS
certificate had expired, so the kubelet couldn't post status to the API server.
On-call cordoned the node, added a NoExecute taint to keep pods off while
poking at it, rotated the cert, saw the node come back Ready, and went to bed.

It's 9 a.m. The node *is* Ready:

```sh
kubectl get nodes
```

```
NAME       STATUS                     ROLES           AGE   VERSION
...        Ready,SchedulingDisabled   <none>          ...
```

…and yet `checkout` is 0/3, all Pending. The incident is over; the *aftermath*
isn't. Read the whole node, not just the STATUS column:

```sh
kubectl describe node <the-node> | grep -A3 -E 'Taints|Unschedulable'
kubectl -n kubelings describe pod -l app=checkout | grep -A4 Events
```

The scheduler's rejection tells you both blockers:
`node(s) were unschedulable` (the cordon) and
`node(s) had untolerated taint {kubelings/maintenance: cert-rotation}`.

## What NotReady actually does (the lifecycle)

"NotReady" isn't a passive label — it's the start of a machine-driven sequence
run by the **node lifecycle controller**:

1. The kubelet posts node status every ~10s. If the control plane hears nothing
   for **40s** (`node-monitor-grace-period`), the node's Ready condition flips
   to `Unknown`/`False` → **NotReady** in `get nodes`.
2. The controller immediately taints the node —
   `node.kubernetes.io/not-ready:NoExecute` (or `unreachable` if status is
   Unknown).
3. Pods don't die instantly: nearly every pod carries an automatic toleration
   for those taints with `tolerationSeconds: 300`. After **5 minutes** still
   NotReady, the taint manager evicts them and controllers reschedule elsewhere.
   (This is taints from Module 5 — same mechanism, applied *by the platform*.)
4. Node recovers → controller **removes its own taints automatically**. What it
   does *not* remove: anything a human added. Cordons and manual taints persist
   until someone deletes them — which is exactly the state you're staring at.

Overnight this node hit steps 1–4; your Pending pods are step 3's eviction plus
step 4's leftovers. On real infrastructure you'd also check *why* it went
NotReady — `kubectl describe node` conditions, then the kubelet's own logs on
the host (`journalctl -u kubelet`): cert expiry, disk full, container runtime
dead, or true network partition.

## Your task

Return the node to service and get `checkout` back to 3/3:

```sh
kubectl get nodes                       # find the SchedulingDisabled one
kubectl describe node <node> | grep Taints
```

<details>
<summary>Hint</summary>

Remove the taint (trailing `-` deletes), then uncordon:

```sh
kubectl taint node <node> kubelings/maintenance:NoExecute-
kubectl uncordon <node>
```

Order matters less than completeness — with only the taint removed the node is
still cordoned; with only the cordon lifted, NoExecute evicts every pod the
scheduler places. Both must go.

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


## Root cause / fix / prevention

- **Root cause:** the *outage* was a kubelet cert expiry; the *lingering
  incident* was human cleanup debt. The platform undoes its own NotReady taints
  when the node recovers; it never touches operator-added cordons and taints.
- **Fix:** `kubectl taint node <n> kubelings/maintenance:NoExecute-` and
  `kubectl uncordon <n>`.
- **Prevention:**
  - Make "uncordon" a runbook *step*, not a memory. Better: end every node
    runbook with `kubectl get nodes | grep SchedulingDisabled` — should be empty.
  - Alert on nodes cordoned > 1h and on `kube_node_spec_taint` for non-platform
    taints — leftover maintenance state is a silent capacity leak.
  - Enable kubelet cert rotation (`rotateCertificates: true`) so this class of
    3 a.m. page stops existing.

## The NotReady triage ladder

For the next one, top down — each step is a lesson you've had:

| Check | Command | If it's this |
|---|---|---|
| Is it one node or many? | `kubectl get nodes` | many → control plane / network (M7), not the node |
| Node conditions | `kubectl describe node` | MemoryPressure/DiskPressure → next lesson |
| Heartbeat age | `lastHeartbeatTime` in conditions | stale → kubelet down or partitioned |
| Platform taints | `not-ready` / `unreachable` present | wait for recovery or fix kubelet on the host |
| Human leftovers | cordon, custom taints | this lesson — clean them up |
| Where did the pods go? | `kubectl get pods -o wide -A` | verify capacity absorbed the evictions |

One design note worth keeping: eviction on NotReady is *deliberately slow*
(40s detection + 300s toleration). That's the platform hedging against flapping
networks — mass-evicting a whole zone because of a 60-second blip would be the
Monzo-style self-amplifying failure you'll study in Module 9. When you tune
`tolerationSeconds` down for faster failover, you're trading that safety margin
away; do it per-workload, not cluster-wide.

</details>
