---
kind: unit
title: "Drill — ghost endpoints after scale-down"
name: pattern-ghost-endpoints-unit
---


> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern
> write-up: [Pattern: ghost endpoints](https://kubelings.madhan.app/incidents/pattern-ghost-endpoints/).

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

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch deploy storefront --type=strategic -p '{
  "spec": {"template": {"spec": {
    "terminationGracePeriodSeconds": 30,
    "containers": [{
      "name": "storefront",
      "lifecycle": {"preStop": {"exec": {"command": ["sh", "-c", "sleep 5"]}}}
    }]
  }}}}'
kubectl -n kubelings rollout status deploy/storefront
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

Pod termination and endpoint removal are **not sequenced** — Kubernetes offers
no "drain, then die" primitive. The contract is: your pod gets a deletion
signal and a grace period; the network catches up *sometime* during it. Teams
hit this as "brief 503s on every deploy", blame the app, add retries, and move
on — until a big scale-down turns the blip into an incident. Setting the grace
period to 0 (impatience) or ignoring SIGTERM (apps that need SIGKILL) makes
the window as wide as possible.

## Fix

```sh
kubectl -n kubelings patch deploy storefront --type=strategic -p '{
  "spec": {"template": {"spec": {
    "terminationGracePeriodSeconds": 30,
    "containers": [{
      "name": "storefront",
      "lifecycle": {"preStop": {"exec": {"command": ["sh", "-c", "sleep 5"]}}}
    }]
  }}}}'
```

The `sleep 5` isn't a hack — it is *deliberately buying time*: the pod stays
up and serving while its IP is scrubbed from every node's routing rules, and
only then gets SIGTERM.

## Prevention / takeaway

- The termination sequence: deletion → (endpoints removal ‖ preStop → SIGTERM
  → grace → SIGKILL). Design for the parallelism.
- Rule of thumb: `preStop sleep N` where N covers your endpoint propagation
  (5s is typical), `terminationGracePeriodSeconds` ≥ N + app shutdown time.
- The app must handle SIGTERM: stop accepting, finish in-flight, exit. The
  Ravelin graceful-shutdown incident in the
  [library](https://kubelings.madhan.app/reference/incident-library/) is this
  drill with a real logo on it.
- `terminationGracePeriodSeconds: 0` has one legitimate user: `kubectl delete
  pod --force` in emergencies. In a template it's a standing outage generator.

</details>
