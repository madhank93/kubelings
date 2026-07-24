---
kind: unit
title: "Incident replay — 502s on every deploy (Ravelin's endpoint secret)"
name: incident-graceful-shutdown-unit
---


## The real incident

**Ravelin**, fraud-detection platform, GKE. Their write-up opens with a
confession every platform team eventually makes: *deploys cause a blip of 502s,
and we'd been ignoring it.*

Source: [Kubernetes' dirty endpoint secret and Ingress — Phil Pearl / Ravelin](https://philpearl.github.io/post/k8s_ingress/)

The mental model everyone has: pod is removed from endpoints → traffic stops →
*then* the pod shuts down. Orderly. Sequential.

The truth Ravelin documented: **those two things happen in parallel.** When a
pod starts terminating:

1. The kubelet sends the container SIGTERM, **and, at the same time,**
2. the endpoints controller starts removing the pod — an update that must then
   propagate to *every* kube-proxy on every node, every ingress controller,
   every cloud LB health check.

Step 1 is one process on one node: milliseconds. Step 2 fans out across the
whole cluster: **seconds**. If the app exits promptly on SIGTERM — like every
well-behaved server — it's gone while nodes are *still routing new requests to
it*. Each one: 502.

The fix is gloriously dumb, and Ravelin says so: **don't die yet. Sleep.** A
preStop hook that waits a few seconds keeps serving while the routing world
catches up, then the app gets SIGTERM and exits clean.

## This cluster, right now

`checkout-api` has the anti-pattern turned up: no preStop, and
`terminationGracePeriodSeconds: 1` — the pod is SIGKILLed one second after
SIGTERM, mid-flight requests be damned. Every rollout is a micro-outage.

## Your task

Make termination outlast endpoint propagation:

1. Add a `preStop` hook that sleeps (~10s is the industry number).
2. Raise `terminationGracePeriodSeconds` to cover preStop **plus** in-flight
   request drain (≥ 15s here).
3. Deployment stays fully Available.

```sh
kubectl -n kubelings get deploy checkout-api -o jsonpath='{.spec.template.spec}' | head -c 400
```

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch deploy checkout-api --type=strategic -p '
spec:
  template:
    spec:
      terminationGracePeriodSeconds: 30
      containers:
        - name: api
          lifecycle:
            preStop:
              exec:
                command: ["sh", "-c", "sleep 10"]
'
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


## The termination timeline (fix in place)

```
t=0     pod marked Terminating
        ├─ endpoints removal begins (fans out cluster-wide, seconds)
        └─ preStop runs: sleep 10 — app STILL SERVING, no SIGTERM yet
t=10s   preStop done → SIGTERM → app drains in-flight, exits
t=30s   grace expires → SIGKILL (only if the app hung)
```

The sleep isn't a hack around Kubernetes — it *is* the mechanism. There is no
"wait until nothing routes to me" signal; the sleep approximates one.

## Layers of the full fix (Ravelin's ladder)

1. **preStop sleep** — covers endpoint propagation. This lesson.
2. **App handles SIGTERM** — stop accepting, drain in-flight, then exit. If your
   server exits instantly on SIGTERM, fix that too.
3. **Grace period ≥ preStop + drain time** — or step 2 gets SIGKILLed anyway.
4. Cloud LBs with slow health-check cycles need the sleep sized to *their*
   propagation, not just kube-proxy's (Ravelin's ingress pain).

## Why this is in the Networking module

It looks like a lifecycle setting, but the failure is a **routing propagation
race** — the same eventual-consistency lesson as endpoints (M1) and conntrack
(previous lesson): *the cluster's view of "who serves traffic" is always
slightly stale. Design shutdown for the staleness.*

## Prevention

- Every HTTP service template ships preStop sleep + honest grace period.
  It's two lines. Codify it.
- Rollout-correlated 5xx spikes in your dashboards = this bug, somewhere.
- Test: `kubectl rollout restart` under load (`hey`/`wrk`) — zero 5xx is
  achievable and should be your bar.

</details>
