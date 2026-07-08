---
kind: unit
title: "Pattern drill: evicted — the disk you forgot to budget"
name: pattern-disk-pressure-unit
---


> **Pattern drill.** Not one company's postmortem — a failure shape that appears
> in almost everyone's: logs, caches, tmp files, and image layers quietly eat
> node disks until the kubelet starts killing things. You'll hit the pod-level
> version live, and learn the node-level version it protects you from.

## The situation

`report-builder` warms a ~200Mi on-disk cache at startup. Watch it for a minute:

```sh
kubectl -n kubelings get pods -w
```

```
report-builder-...   1/1   Running   0     45s
report-builder-...   0/1   Evicted   0     52s
report-builder-...   1/1   Running   0     4s
```

**Evicted** is a status you haven't met. It isn't a crash — the process didn't
exit, there's no restart count, no CrashLoopBackOff. The **kubelet** killed the
pod, and the corpse stays behind with the reason written on it:

```sh
kubectl -n kubelings describe pod <an-evicted-one> | grep -B2 -A4 Evicted
```

```
Status:  Failed
Reason:  Evicted
Message: Pod ephemeral local storage usage exceeds the total limit
         of containers 64Mi.
```

Everything a container writes to its own filesystem — plus `emptyDir` volumes
and its logs — is **ephemeral storage**, drawn from the node's disk. It's the
fourth resource after CPU, memory, GPU… and the one nobody declares. This pod
has a 64Mi limit and a 200Mi appetite; the kubelet enforces the limit by
eviction. Then the ReplicaSet — which only counts *running* replicas — replaces
it, the warmup starts over, and you get an eviction treadmill that looks
almost healthy from a distance (`1/2 available`, forever).

Two things memory taught you that do **not** carry over:

- Memory overuse ⇒ OOMKill ⇒ *restart in place* (Module 2). Disk overuse ⇒
  **eviction** ⇒ pod deleted, rescheduled from scratch.
- OOMKilled pods vanish into restart counts; **Evicted pods linger as Failed
  objects** until a human (or a tuned GC threshold) deletes them.

## Your task

1. Budget the real footprint: raise the container's `ephemeral-storage` limit
   so a ~200Mi warmup fits with headroom (≥ 256Mi is honest).
2. Wait out one clean warmup — both replicas Running past the old kill line.
3. Bury the corpses: delete the Evicted (Failed) pods so `get pods` tells the
   truth again.

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch deploy report-builder --type=json -p '[
  {"op":"replace",
   "path":"/spec/template/spec/containers/0/resources/limits/ephemeral-storage",
   "value":"512Mi"}]'
kubectl -n kubelings delete pod --field-selector=status.phase=Failed
```

The warmup takes ~45s after the new pods start; the verify waits for full
availability.

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


## The node-level version (the one that pages you)

What you just debugged was the *contained* failure: a pod exceeding **its own**
limit gets evicted, blast radius one pod. Remove the limit and the failure
changes shape: the pod eats the **node's** disk until the kubelet crosses its
eviction threshold (nodefs < 10% free by default), sets the node condition
`DiskPressure`, and the platform responds:

1. `node.kubernetes.io/disk-pressure:NoSchedule` taint appears — new pods stop
   landing there (the condition-taint mechanism from the NotReady lesson).
2. The kubelet reclaims: first **image garbage collection** (unused layers),
   then **pod eviction** — and here your QoS class from Module 2 becomes a
   death order: pods *exceeding their requests* go first, BestEffort before
   Burstable before Guaranteed.
3. Innocent pods on the node die for one tenant's log spew — the noisy-neighbor
   pattern from Module 5, on the disk axis.

So the per-pod limit you set isn't bureaucracy: it converts "node-wide
DiskPressure, victims chosen by QoS" into "the guilty pod gets evicted, alone."

## Root cause / fix / prevention

- **Root cause:** disk treated as free and infinite — a 200Mi working set
  behind a 64Mi limit (and, in the wild, usually behind *no* limit on a shared
  node).
- **Fix:** measure the real footprint, set `ephemeral-storage` requests *and*
  limits like you do memory; clean up Failed pods.
- **Prevention:**
  - Requests matter, not just limits: the scheduler uses `ephemeral-storage`
    *requests* to avoid stacking disk-hungry pods on one node.
  - Big scratch data belongs in a sized volume — `emptyDir` with `sizeLimit`,
    or a real PVC (Module 3) — not the container filesystem.
  - Logs: rotate in-app or cap them; container logs count against the node,
    and "we logged the node to death" is a genuinely common postmortem.
  - Alert on `kubelet_evictions` and on node `DiskPressure` transitions; a
    node that flaps in and out of DiskPressure is one bad deploy from a page.
  - `kubectl get pods -A --field-selector=status.phase=Failed` in your weekly
    hygiene — eviction corpses are free forensic evidence, then noise.

</details>
