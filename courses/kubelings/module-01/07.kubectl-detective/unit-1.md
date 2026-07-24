---
kind: unit
title: "kubectl detective: find the broken one"
name: kubectl-detective-unit
---


## The situation

The alert reads, in its entirety: **"1 of 5 services degraded."**

Which one? The alert doesn't say. The dashboard that would say is owned by a team
in another timezone. You have `kubectl`, the `kubelings` namespace, and five
deployments: `catalog`, `checkout`, `payments`, `search`, `recommendations`.

This is the daily reality of cluster operations: not exotic failures, but *"which
of these many identical-looking things is the broken one?"* The engineers who
resolve these in minutes aren't luckier — they run the same short loop every
time, widest view first, narrowing on anomaly:

```
get (wide, everything) → spot the odd number → describe it → events/logs → fix
```

## Your task

Find the degraded service and restore it. Every deployment must end with at
least one available replica.

Start wide:

```sh
kubectl -n kubelings get deploy          # READY column — read it like a detective
kubectl -n kubelings get pods
kubectl -n kubelings get events --sort-by=.lastTimestamp | tail -15
```

<details>
<summary>Hint</summary>

`get deploy` shows READY `n/m`. Four say `1/1`. One says `0/0` — zero *desired*.
Nobody's pod is crashing; someone scaled it to zero and forgot. Scale it back:

```sh
kubectl -n kubelings scale deploy/recommendations --replicas=1
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


## Root cause

`recommendations` was scaled to `0` — a maintenance action never rolled back.
The sneaky part: **nothing is red.** No crashloop, no failed pods, no error
events. A deployment at 0/0 is *satisfied* — Kubernetes is doing exactly what it
was told. The only signal is a number that doesn't match its neighbors.

## The detective's loop

```sh
kubectl -n kubelings get deploy                      # 0/0 ≠ 1/1 — anomaly found
kubectl -n kubelings scale deploy/recommendations --replicas=1
kubectl -n kubelings get deploy                      # 1/1 everywhere
```

Wide first (`get` across the namespace), then narrow (`describe` one object),
then history (`events`, `logs`). Resist the urge to describe things one by one
from the top — the wide view tells you *where* to look.

## Worth memorizing

```sh
kubectl get deploy,sts,ds -A                       # fleet health in one line
kubectl get pods -A --field-selector=status.phase!=Running
kubectl get events -A --sort-by=.lastTimestamp | tail -20
```

## Prevention

- Alert on `spec.replicas == 0` for services that should never be zero.
- Scale-downs for maintenance get a ticket with an expiry, or better, a
  `kubectl rollout restart` instead where possible.

</details>
