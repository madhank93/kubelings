---
kind: unit
title: "QoS classes: who gets killed first"
name: qos-classes-unit
---


## The situation

A node runs out of memory. This is not hypothetical — it happens in every fleet,
every week. The kernel must reclaim memory *now*, and something must die.

Kubernetes doesn't flip a coin. Every pod gets a **QoS class** at creation,
computed silently from its resources — and that class is the kill order:

| Class | Rule | Under pressure |
|---|---|---|
| **Guaranteed** | every container: requests == limits (cpu *and* memory) | evicted last |
| **Burstable** | at least one request set, but not Guaranteed | middle |
| **BestEffort** | no requests, no limits anywhere | **evicted first** |

Your namespace tonight:

- `pay-core` — the payment service. Has requests *and* limits, but they differ →
  **Burstable**. One tier above the chopping block is not where payments belongs.
- `batch-reindex` — an overnight job with **no resources declared at all** →
  **BestEffort**. Worse: with no requests, the scheduler thinks it's *free* and
  packs it anywhere, which is exactly how nodes end up under pressure.

```sh
kubectl -n kubelings get pods -o custom-columns=NAME:.metadata.name,QOS:.status.qosClass
```

## Your task

1. Make `pay-core` **Guaranteed**: requests == limits for cpu *and* memory.
2. Make `batch-reindex` honest: give it real requests (Burstable is fine —
   batch work *should* be sacrificed before payments).
3. Both deployments stay Available.

<details>
<summary>Hint</summary>

```sh
# Guaranteed = requests and limits identical:
kubectl -n kubelings set resources deploy/pay-core \
  --requests=cpu=200m,memory=128Mi --limits=cpu=200m,memory=128Mi

# Honest batch (requests set, limits above them -> Burstable):
kubectl -n kubelings set resources deploy/batch-reindex \
  --requests=cpu=50m,memory=32Mi --limits=cpu=200m,memory=128Mi
```

QoS is immutable per pod — the rollout replaces pods, and the *new* ones carry
the new class. Check with the custom-columns command above.

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


## Why this ordering exists

Requests are a *promise you asked for*; limits are a *ceiling you accepted*.

- Guaranteed pods never use more than they reserved — evicting them buys the
  node nothing it wasn't already promised away. So they go last.
- BestEffort pods reserved nothing — every byte they hold is squatted. They go
  first.
- Burstable pods are ranked among themselves by *how far above their request*
  they're actually using — the biggest overachiever dies first.

## Fix

```sh
kubectl -n kubelings set resources deploy/pay-core \
  --requests=cpu=200m,memory=128Mi --limits=cpu=200m,memory=128Mi
kubectl -n kubelings set resources deploy/batch-reindex \
  --requests=cpu=50m,memory=32Mi --limits=cpu=200m,memory=128Mi
kubectl -n kubelings get pods -o custom-columns=NAME:.metadata.name,QOS:.status.qosClass
```

## The double sin of no-requests

`batch-reindex` with nothing declared wasn't just first to die — it was
**invisible to the scheduler**. Requests are what scheduling *is*: the scheduler
packs nodes by summing requests. A zero-request pod fits "anywhere," lands on
busy nodes, and manufactures the very memory pressure that then kills it (and
its neighbors). Declaring requests is citizenship, not paperwork.

## Prevention

- Payments-tier: Guaranteed, always. Batch-tier: Burstable with honest requests.
  BestEffort: reserve for truly disposable experiments — or ban it with a
  LimitRange that injects defaults.
- Audit: `kubectl get pods -A -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name,QOS:.status.qosClass | grep BestEffort`
- This ordering is exactly what you'll watch happen live in Module 8's eviction
  lesson.

</details>
