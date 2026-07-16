> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern

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
