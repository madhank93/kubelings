---
kind: unit
title: "Fix the Rolling Update: unsafe maxSurge/maxUnavailable"
name: rolling-update-unit
---


## The situation

The `web` Deployment in the `kubelings` namespace serves production traffic, but
every release causes a brief total outage. Its `RollingUpdate` strategy is set to
`maxSurge: 0` and `maxUnavailable: 100%` — so on each rollout Kubernetes is allowed
to terminate **all** pods before any replacement is Ready, and is **not** allowed
to surge a single extra pod to cover the gap.

## Your task

Make `web`'s rolling update **zero-downtime**:

1. Set `maxSurge` so at least one new pod can start before old ones go away.
2. Set `maxUnavailable` so the whole fleet can't be taken down at once.
3. Keep the Deployment Available (all 3 replicas Ready).

```sh
kubectl -n kubelings get deploy web -o yaml | less
```

<details>
<summary>Hint</summary>

Edit the strategy, e.g.:

```sh
kubectl -n kubelings patch deploy web --type=merge -p \
  '{"spec":{"strategy":{"rollingUpdate":{"maxSurge":1,"maxUnavailable":"25%"}}}}'
```

`maxSurge` ≥ 1 lets a replacement come up first; `maxUnavailable` < 100% keeps
capacity during the roll.

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

`web`'s `RollingUpdate` strategy was `maxSurge: 0` + `maxUnavailable: 100%`. That
combination lets a rollout delete every pod at once (100% unavailable) while
forbidding any extra pod from starting first (0 surge) — a guaranteed outage on
every deploy.

## Fix

```sh
kubectl -n kubelings patch deploy web --type=merge -p \
  '{"spec":{"strategy":{"rollingUpdate":{"maxSurge":1,"maxUnavailable":"25%"}}}}'
```

or `kubectl -n kubelings edit deploy web` and set:

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 25%
```

## Verify

```sh
kubectl -n kubelings rollout status deploy/web
kubectl -n kubelings get deploy web \
  -o jsonpath='{.spec.strategy.rollingUpdate}{"\n"}'
```

</details>
