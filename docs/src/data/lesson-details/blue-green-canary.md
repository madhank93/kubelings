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
