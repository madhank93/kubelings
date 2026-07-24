---
kind: unit
title: "Drill — the rolling update that deadlocks"
name: pattern-rolling-update-deadlock-unit
---


> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern
> write-up: [Pattern: rolling update deadlock](https://kubelings.madhan.app/incidents/pattern-rolling-update-deadlock/).

## The situation

The v2 release of `inventory` went out an hour ago. It is still "going out":

```sh
kubectl -n kubelings rollout status deploy/inventory
# Waiting for deployment "inventory" rollout to finish: 1 old replicas are pending termination...
```

Nothing is crashing. Old pods: `Running`. New pod: `Pending`. Forever:

```sh
kubectl -n kubelings get pods -l app=inventory
kubectl -n kubelings get deploy inventory -o jsonpath='{.spec.strategy}'
```

The strategy says `maxUnavailable: 0, maxSurge: 1` — a zero-downtime policy:
never drop below 2 Ready replicas, roll by surging 1 extra pod. Perfectly
reasonable. But the surge pod can't schedule:

```sh
kubectl -n kubelings describe pod -l app=inventory | grep -A3 Events | tail -5
# 0/N nodes are available: insufficient cpu.
```

The v2 template requests **64 CPUs** — a resource block copy-pasted from
another environment. Now trace the deadlock:

- No old pod may terminate until a new pod is Ready (`maxUnavailable: 0`).
- The one allowed new pod (`maxSurge: 1`) can never be Ready — no node fits it.
- The Deployment controller is not stuck on a bug. It is *correctly* refusing
  to violate your own availability policy. It will wait for weeks.

(Why not `maxSurge: 0` too? The API rejects it — both zero would make progress
literally impossible, so validation forbids the combination.)

## Your task

Ship v2 — fix the *cause*, keep the policy:

1. Find why the new pod is Pending (`kubectl describe` its Events).
2. Fix the v2 resource request to something a node can host.
3. `kubectl rollout undo` also "unsticks" it — but that cancels the release;
   the check requires v2 (`VERSION=v2` env) to actually ship.

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch deploy inventory --type=json -p '[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests",
   "value": {"cpu": "10m", "memory": "16Mi"}}
]'
kubectl -n kubelings rollout status deploy/inventory
```

Patching the template creates revision 3 — same v2 env, sane request — and the
rollout completes without ever dropping below 2 Ready pods.

</details>

::simple-task
---
:tasks: tasks
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>


## The pattern (why this recurs everywhere)

Two individually-correct decisions compose into a deadlock:

1. **`maxUnavailable: 0`** — the SRE-approved zero-downtime strategy.
2. **A new revision that cannot become Ready** — oversized requests (this
   drill), an unpullable image, a failing readiness probe, a missing PVC, an
   unsatisfiable nodeSelector… anything.

With headroom (`maxUnavailable: 1`) the same bad revision would at least swap
one pod and *visibly* fail. With zero headroom it fails silently — dashboards
green, old version serving, release "in progress" for days. Teams usually
discover it when the *next* release queues up behind this one.

## Fix

```sh
# diagnose: the Pending pod names the blocker
kubectl -n kubelings describe pod -l app=inventory | grep -B2 -A6 Events

# fix the cause — sane v2 request (creates revision 3, still VERSION=v2):
kubectl -n kubelings patch deploy inventory --type=json -p '[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests",
   "value": {"cpu": "10m", "memory": "16Mi"}}
]'
kubectl -n kubelings rollout status deploy/inventory   # completes
```

`rollout undo` is the right move when v2 itself is bad and users are hurting —
but here v2 was fine, its resource block wasn't, and undo would just re-queue
the same deadlock for the next attempt.

## Prevention / takeaway

- `maxUnavailable: 0` **requires** `maxSurge ≥ 1` (API-enforced) — which means
  it requires *schedulable headroom* for the surge pod. Zero-downtime rollouts
  are a capacity commitment, not just a YAML flag.
- Set `spec.progressDeadlineSeconds` (default 600) and **alert on
  `Progressing=False` / reason `ProgressDeadlineExceeded`** — that's the
  signal this drill's victims were missing for hours.
- Resource requests are per-environment config, not portable constants —
  the 64-CPU copy-paste is this pattern's most common trigger in the wild
  (M2.14's CPU-throttling incident is the same mistake in the other
  direction).
- Watch rollouts to completion in CI/CD (`kubectl rollout status --timeout`)
  instead of fire-and-forget `kubectl apply`.

</details>
