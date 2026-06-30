# Solution — safe rolling update strategy

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
