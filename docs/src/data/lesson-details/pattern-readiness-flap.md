> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern

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
