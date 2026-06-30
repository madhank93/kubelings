# Solution — right-size memory limits

## Root cause

The `cache` container needs ~50Mi but `limits.memory` was `20Mi`. When a container
exceeds its memory limit the kernel OOM-kills it, producing
`Last State: Terminated / Reason: OOMKilled` and CrashLoopBackOff.

## Fix

Raise the request and limit above the working-set size (with headroom):

```sh
kubectl -n kubelings set resources deploy/cache \
  --requests=memory=64Mi --limits=memory=128Mi
```

or `kubectl -n kubelings edit deploy cache` and set:

```yaml
resources:
  requests: {memory: 64Mi}
  limits:   {memory: 128Mi}
```

## Verify

```sh
kubectl -n kubelings rollout status deploy/cache
kubectl -n kubelings get pods -l app=cache   # Running, RESTARTS stays 0
```

## Prevention

Set memory `requests` from observed usage and `limits` with headroom; alert on
`container_memory_working_set_bytes` approaching the limit before pods OOM.
