---
kind: unit
title: "Drill — the PVC stuck Terminating"
name: pattern-pvc-terminating-unit
---


> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern
> write-up: [Pattern: PVC stuck Terminating](https://kubelings.madhan.app/incidents/pattern-pvc-terminating/).

## The situation

A teardown job is retiring an old environment. Everything deletes cleanly —
except:

```
NAME       STATUS        VOLUME     CAPACITY   ACCESS MODES   AGE
data-old   Terminating   pvc-4f2…   1Gi        RWO            94d
```

An hour now. CI is timing out. It's 2 a.m. somewhere and a voice in the incident
channel suggests: *"just strip the finalizers, I found a command on
StackOverflow."*

Do not. First, understand what's actually happening — because **nothing is
stuck**. Look:

```sh
kubectl -n kubelings get pvc data-old -o jsonpath='{.metadata.finalizers}'
```

```
["kubernetes.io/pvc-protection"]
```

Deletion in Kubernetes is a two-phase protocol: the object gets a
`deletionTimestamp` (that's your `Terminating`), then each **finalizer** — a
controller's registered veto — must be satisfied and removed before the object
actually goes. `pvc-protection`'s condition is simple: **no pod may still be
using this claim.** Some pod still is. The system is refusing to yank a disk out
from under a running process. That's not a bug. That's the seatbelt.

## Your task

Release the claim *properly*:

1. Find who still mounts `data-old` (`describe pvc` has a `Used By:` field).
2. Deal with the consumer.
3. Watch the PVC delete **itself** — you never touch the finalizer.

```sh
kubectl -n kubelings describe pvc data-old | grep -i "used by"
```

The check fails you if you strip finalizers. The seatbelt stays on.

<details>
<summary>Hint</summary>

`Used By: debug-shell-leftover` — someone's forgotten debugging pod from last
quarter. Remove it and the finalizer releases within seconds:

```sh
kubectl -n kubelings delete pod debug-shell-leftover
kubectl -n kubelings get pvc    # data-old gone
```

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


## The pattern (why this recurs everywhere)

Ghost consumers accumulate in every long-lived cluster: paused debug pods,
completed-but-retained Job pods, CronJob runs beyond their history limit. Months
later a teardown hits one, and "delete hangs" pages a human. The postmortem is
always the same sentence: *a finalizer was waiting on something nobody
remembered existed.*

## Fix

```sh
kubectl -n kubelings describe pvc data-old | grep -i "used by"
kubectl -n kubelings delete pod debug-shell-leftover
```

## Why finalizer-stripping corrupts

`kubectl patch pvc data-old -p '{"metadata":{"finalizers":null}}'` "works" — the
object vanishes. But the pod keeps running, kernel file handles open, writing to
a volume whose claim no longer exists. The PV's reclaim policy may fire and
delete the backing disk **while data is in flight**. You've converted a visible
hang into invisible data loss. Reserve it for objects whose controller is
*permanently gone* (uninstalled CRDs) — never for working protection.

## Prevention

- Teardown order: workloads → PVCs → PVs. Encode it in the pipeline.
- `ttlSecondsAfterFinished` on Jobs; history limits on CronJobs — ghost-consumer
  birth control.
- Alert on any object `Terminating` > 10 min: it always means a finalizer is
  waiting on something you can name with one `describe`.

</details>
