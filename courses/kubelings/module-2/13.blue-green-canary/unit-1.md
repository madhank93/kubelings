---
kind: unit
title: "Blue/green: the deploy you can undo in one second"
name: blue-green-canary-unit
---


## The situation

Release night for `shop` v2. Last month's v2-attempt used a rolling update
(lesson 2.1) — and when v2 turned out to corrupt carts, "rollback" meant
*another* rolling update backwards, ten anxious minutes of mixed v1/v2 traffic
both ways. The postmortem action item: **the next flip must be instant, and so
must the un-flip.**

So this time the team prepared blue/green:

```sh
kubectl -n kubelings get deploy,svc -l app=shop --show-labels
kubectl -n kubelings get svc shop -o jsonpath='{.spec.selector}'
```

Two parallel deployments — `shop-blue` (v1, serving) and `shop-green` (v2,
**scaled to zero**) — and one Service whose selector contains the switch:

```
{"app":"shop","track":"blue"}
```

You've seen everything here before, pointed at a new purpose: the Service
selector (lesson 1.4's mismatch, used *deliberately*), readiness probes
(lesson 2.8) as the flip gate, and Endpoints as the proof.

## Your task

Ship v2 without dropping a request, keeping the escape hatch:

1. Scale `shop-green` to 2 and wait until both pods are **Ready** — the
   readiness probe is what makes "green is up" a fact instead of a hope.
2. Flip the Service: `track: blue` → `track: green`. Endpoints repopulate in
   about a second; conntrack keeps established connections alive while new
   ones go to green.
3. **Leave `shop-blue` running.** Until v2 has soaked, blue is not cruft —
   it's your one-second rollback.

```sh
kubectl -n kubelings get endpoints shop    # watch the flip land
```

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings scale deploy shop-green --replicas=2
kubectl -n kubelings rollout status deploy/shop-green
kubectl -n kubelings patch svc shop --type=merge \
  -p '{"spec":{"selector":{"app":"shop","track":"green"}}}'
```

Rollback rehearsal (don't leave it this way): the same patch with
`"track":"blue"`. One second, either direction.

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


## The three strategies, one table

| | rolling update | blue/green | canary |
|---|---|---|---|
| how | replace pods in place | two stacks, flip selector | two stacks, *share* the selector |
| cutover | gradual, minutes | instant | gradual, *controlled by you* |
| rollback | reverse rollout (slow) | flip back (instant) | scale canary to 0 (instant) |
| cost | ~1× | **2×** while both run | 1× + canary |
| catches | crashes at startup | everything green testing catches | **real-traffic** regressions |

**Canary with what you have:** keep both deployments matching the Service
(selector `app: shop` only, drop `track`), and set replicas 9:1 blue:green —
kube-proxy spreads connections roughly evenly across endpoints, so ~10% of
traffic hits v2. Watch error rates, then shift the ratio. That's the poor
man's canary; the rich man's (header-based routing, 1% by *request*, auto
rollback on metrics) needs an Ingress controller or mesh doing L7 — the same
Ingress you'll wire in Module 4.

## What blue/green actually costs

- **Double capacity** while both stacks run — quota (M8.3) and cluster
  headroom must fit 2× during the window. Plan the flip, don't camp on it.
- **State**: two app versions against one database means your schema must
  tolerate both — expand/contract migrations. The selector flip is easy;
  *data* is why blue/green is a discipline, not a trick.
- **Sessions**: the flip strands in-memory session state on blue. Externalize
  state, or drain: flip, keep blue until connections age out, then scale down.

## Root cause / fix / prevention (of the original bad night)

- **Root cause:** rolling update as the *only* deploy path — no instant
  rollback when v2 misbehaved under real traffic.
- **Fix:** parallel stacks + selector flip; readiness-gated, endpoint-verified.
- **Prevention:** rehearse the rollback flip in the same change window you
  rehearse the deploy; alert on Endpoints count for the Service so a flip to
  a not-ready stack pages in seconds; automate the soak-then-scale-down of
  the old color so 2× cost doesn't become permanent.

</details>
