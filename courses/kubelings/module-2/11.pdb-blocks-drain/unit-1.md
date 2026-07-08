---
kind: unit
title: "The drain that never finishes"
name: pdb-blocks-drain-unit
---


## The situation

Kernel patch night. Runbook step 3: `kubectl drain kubelings-worker`. You run it
and watch:

```
evicting pod kubelings/tickets-6d8f7b9c4-2xkpl
error when evicting pods/"tickets-6d8f7b9c4-2xkpl" -n "kubelings" (will retry after 5s):
Cannot evict pod as it would violate the pod's disruption budget.
```

…and again. And again. Ten minutes of the same two lines. The drain isn't
broken — it's being *refused*, politely, forever.

A **PodDisruptionBudget** is a contract with the eviction API: *"never let
voluntary disruptions take availability below X."* Drains, node upgrades,
cluster-autoscaler scale-downs — all go through eviction and all must honor it.

Now the math on this one:

```
replicas:      2
minAvailable:  2
disruptionsAllowed = 2 - 2 = 0
```

Zero. Not "wait until it's safer" — **structurally zero, at all times**. Whoever
wrote this PDB demanded 100% of replicas be up always, which outlaws
maintenance itself.

## Your task

Make maintenance possible without giving up protection:

1. Look at what the PDB currently allows (`kubectl get pdb` shows the columns).
2. Fix the contract so at least one disruption is allowed while the app stays
   protected — change the PDB's math, or give it more replicas to budget with.
   Keep a PDB either way.
3. `tickets` must remain fully Available.

```sh
kubectl -n kubelings get pdb tickets-pdb
kubectl -n kubelings get deploy tickets
```

<details>
<summary>Hint</summary>

Two honest fixes:

```sh
# a) express the budget as tolerated disruption:
kubectl -n kubelings patch pdb tickets-pdb --type=merge \
  -p '{"spec":{"minAvailable":null,"maxUnavailable":1}}'

# b) or fund the budget with headroom:
kubectl -n kubelings scale deploy/tickets --replicas=3   # minAvailable:2 now allows 1
```

Then check: `kubectl -n kubelings get pdb` → ALLOWED DISRUPTIONS ≥ 1.

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

`minAvailable: 2` on a 2-replica deployment = `disruptionsAllowed: 0`. The
eviction API did exactly its job: it refused every eviction, indefinitely. In
real fleets this is how "one team's PDB" blocks *cluster-wide* node upgrades —
the autoscaler and upgrade tooling retry forever against one immovable budget.

## Fix

Either express tolerance:

```sh
kubectl -n kubelings patch pdb tickets-pdb --type=merge \
  -p '{"spec":{"minAvailable":null,"maxUnavailable":1}}'
```

or fund the budget:

```sh
kubectl -n kubelings scale deploy/tickets --replicas=3
```

Both end with `ALLOWED DISRUPTIONS: 1` — drain proceeds one pod at a time,
availability floor intact.

## PDB math cheat sheet

| replicas | PDB | allowed disruptions |
|---|---|---|
| 2 | `minAvailable: 2` | **0 — drains hang** |
| 3 | `minAvailable: 2` | 1 |
| 2 | `maxUnavailable: 1` | 1 |
| 1 | any PDB | 0 — singletons can't be protected *and* drainable |

Prefer `maxUnavailable` — it keeps allowing disruption as you scale up, whereas
`minAvailable` as an absolute number silently tightens when replicas shrink.

## Prevention

- CI-lint PDBs against replica counts: `disruptionsAllowed == 0` at steady state
  is a bug, not a policy.
- Involuntary disruptions (node crash, OOM) ignore PDBs entirely — a PDB is not
  HA; it only shapes *voluntary* churn.
- Before maintenance windows: `kubectl get pdb -A` and look at the ALLOWED
  column. Zeroes = tonight's incident.

</details>
