---
kind: unit
title: "Drill — the readiness probe that flaps"
name: pattern-readiness-flap-unit
---


> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern
> write-up: [Pattern: readiness flapping](https://kubelings.madhan.app/incidents/pattern-readiness-flap/).

## The situation

Users report intermittent 502s from `search-api`. The pods never crash, never
restart — but watch them for thirty seconds:

```sh
kubectl -n kubelings get pods -l app=search-api -w
```

```
search-api-7d9f…   1/1   Running   0   2m
search-api-7d9f…   0/1   Running   0   2m
search-api-7d9f…   1/1   Running   0   2m
search-api-7d9f…   0/1   Running   0   2m
```

`READY` flickers like a loose bulb. Every flip to `0/1` removes the pod from
the Service's endpoint pool; every flip back re-adds it. Requests in flight
during a removal get 502s. Check the endpoints churn:

```sh
kubectl -n kubelings get endpoints search-api -w
kubectl -n kubelings describe pod -l app=search-api | grep -A4 "Readiness"
```

The probe: `periodSeconds: 1`, `failureThreshold: 1`, and a health check that
blips roughly one second in three under load. One blip = instant eviction from
the pool. The app is *fine* — the probe is a hair trigger.

## Your task

Tune the readiness probe on the `search-api` Deployment so single blips no
longer evict pods:

- `failureThreshold` ≥ 3 — it takes *consecutive* failures to unready a pod
- `periodSeconds` ≥ 5 — probing every second amplifies every hiccup

Deployments are mutable — `kubectl edit deploy` or a JSON patch both work.
Don't delete the probe: a Service with no readiness signal happily routes to
booting pods, which is a different outage.

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch deploy search-api --type=json -p '[
  {"op": "replace", "path": "/spec/template/spec/containers/0/readinessProbe/failureThreshold", "value": 3},
  {"op": "replace", "path": "/spec/template/spec/containers/0/readinessProbe/periodSeconds", "value": 5}
]'
kubectl -n kubelings rollout status deploy/search-api
```

The blip still happens — but it's never 3 in a row, so the pod never leaves
the pool.

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

Readiness probes get copy-pasted from liveness examples or tightened "to react
faster" — and a probe that reacts in 1 second to a 1-second blip converts every
transient GC pause, connection-pool wait, or noisy-neighbor CPU spike into
endpoint churn. Under real load, blips are *normal*. The probe's job is to
detect "not serving", not "momentarily slow".

## Fix

```sh
kubectl -n kubelings patch deploy search-api --type=json -p '[
  {"op": "replace", "path": "/spec/template/spec/containers/0/readinessProbe/failureThreshold", "value": 3},
  {"op": "replace", "path": "/spec/template/spec/containers/0/readinessProbe/periodSeconds", "value": 5}
]'
```

With `failureThreshold: 3` the pod must fail **three consecutive** probes
(~15 s of genuine unhealth) before eviction. The 1-in-3 blip never lands three
in a row — pods hold steady while a real outage still gets caught.

## Verify

```sh
kubectl -n kubelings get pods -l app=search-api        # 1/1, 1/1 — and staying there
kubectl -n kubelings get endpoints search-api          # both IPs, stable
```

## Prevention / takeaway

- **Readiness and liveness have opposite failure costs.** A slow readiness
  probe delays traffic; a slow liveness probe delays a restart. A *twitchy*
  readiness probe causes 502 churn; a twitchy liveness probe causes restart
  storms. Tune them separately.
- Rules of thumb: readiness `periodSeconds: 5–10`, `failureThreshold: 3`;
  add `initialDelaySeconds` (or better, a `startupProbe` — M2.9) for slow
  boots.
- Alert on endpoint churn rate, not just pod restarts — flapping is invisible
  to restart counters.
- The probed path must be cheap and dependency-free: a probe that calls your
  database turns database blips into *your* outage (M8's cascading-failure
  material picks this up).

</details>
