> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern

## The situation

Every deploy of `storefront` — and every scale-down — produces a five-second
burst of 503s, then everything is fine again. Nobody can catch it in the act;
by the time a human looks, the endpoint list is correct.

The bug is in the *timeline* of pod termination. When a pod is deleted, two
things start **in parallel**:

1. The kubelet runs the `preStop` hook (if any), then sends SIGTERM, waits up
   to `terminationGracePeriodSeconds`, then SIGKILL.
2. The endpoints controller removes the pod's IP from the Service — and every
   kube-proxy on every node updates its rules *eventually*.

Track 2 takes real time to propagate. If track 1 finishes first, traffic is
still being routed to an IP whose process is already dead: **ghost
endpoints**. Now look at this Deployment:

```sh
kubectl -n kubelings get deploy storefront -o jsonpath='{.spec.template.spec.terminationGracePeriodSeconds}'
# 0
kubectl -n kubelings get deploy storefront -o jsonpath='{.spec.template.spec.containers[0].lifecycle}'
# (nothing)
```

`terminationGracePeriodSeconds: 0` — the pod is SIGKILLed the *instant*
deletion begins, guaranteeing a window where the routing mesh points at a
corpse. Someone set it "to make deploys faster".

## Your task

Fix the `storefront` Deployment's termination timeline:

1. Add a `preStop` hook that keeps the process alive while the endpoint
   removal propagates — the classic is a short sleep:

   ```yaml
   lifecycle:
     preStop:
       exec:
         command: ["sh", "-c", "sleep 5"]
   ```

2. Give the grace period room to cover it: `terminationGracePeriodSeconds`
   ≥ 5 (or delete the field — the default is 30).

Deployments are mutable; `kubectl edit` or `kubectl patch` both roll the
change out.
