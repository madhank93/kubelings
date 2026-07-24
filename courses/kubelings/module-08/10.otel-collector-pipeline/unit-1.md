---
kind: unit
title: "OTel pipeline: traces into the void"
name: otel-collector-pipeline-unit
---


## The situation

Three deployments form a textbook tracing pipeline:

```
span-source (telemetrygen) ──OTLP──▶ otel-collector ──OTLP──▶ tracing-backend (Jaeger)
```

The app sends spans, the **OpenTelemetry Collector** receives them, and
Jaeger stores and serves them. Except Jaeger is empty:

```sh
kubectl get --raw "/api/v1/namespaces/kubelings/services/tracing-backend:16686/proxy/api/services"
# {"data":[],"total":0,…}
```

The collector is Running, 1/1 Ready — and quietly furious:

```sh
kubectl -n kubelings logs deploy/otel-collector --tail=20
# … "Exporting failed. Will retry the request after interval." …
# … name resolver error: produced zero addresses …
```

A collector pipeline is three lists in a ConfigMap — `receivers` →
`processors` → `exporters`, wired together under `service.pipelines`:

```sh
kubectl -n kubelings get configmap otel-collector-config -o jsonpath='{.data.config\.yaml}'
```

The exporter says `endpoint: jaeger-collector.observability:4317` — a
service name from some *other* cluster's setup, pasted in with the config.
Nothing in this namespace answers to it:

```sh
kubectl -n kubelings get svc
# otel-collector, span-source?, tracing-backend   ← there's your backend
```

Meanwhile the collector accepts every span from the app (the receiver
works!), buffers, retries, and eventually drops them. **The pipeline fails
in the middle, invisibly to both ends** — the app sees successful sends,
Jaeger sees nothing. That's why collector logs, not app logs, are where
tracing outages live.

## Your task

1. Fix the exporter endpoint in the ConfigMap to the Service that actually
   fronts Jaeger here — `tracing-backend:4317`:

   ```sh
   kubectl -n kubelings edit configmap otel-collector-config
   ```

2. **Restart the collector.** ConfigMap edits do not restart anything
   (M3.9's lesson: mounted files sync, but the collector reads config once
   at startup):

   ```sh
   kubectl -n kubelings rollout restart deploy/otel-collector
   kubectl -n kubelings rollout status deploy/otel-collector
   ```

3. Watch the logs go quiet, then ask Jaeger who's reporting:

   ```sh
   kubectl -n kubelings logs deploy/otel-collector --tail=10   # no export errors
   kubectl get --raw "/api/v1/namespaces/kubelings/services/tracing-backend:16686/proxy/api/services"
   # {"data":["checkout"],…}
   ```

<details>
<summary>Hint</summary>

Full DNS form works too: `tracing-backend.kubelings.svc:4317`. The check
accepts either — what matters is that the name resolves to the Jaeger
Service (M4's service-DNS lessons, cashed in).

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


## Fix

```sh
kubectl -n kubelings get cm otel-collector-config -o yaml \
  | sed 's|jaeger-collector.observability:4317|tracing-backend:4317|' \
  | kubectl apply -f -
kubectl -n kubelings rollout restart deploy/otel-collector
kubectl -n kubelings rollout status deploy/otel-collector
```

## Why a collector at all

The app could export straight to Jaeger — until you have twelve apps, a
backend migration, sampling decisions, and an outage where telemetry
floods the network. The collector is the indirection layer: apps speak
OTLP to one local address forever; batching, retry, sampling, and *where
telemetry actually goes* become ops config, not app redeploys. This lesson
is the cost of that power: the collector is now a hop that can silently
eat everything (Datadog/M8.6's "observer must not share fate with the
observed" applies — in production the collector gets its own health
alerts, via its telemetry endpoints).

## Prevention / takeaway

- **Debug pipelines middle-out**: receiver counters up? exporter errors?
  The collector's own logs and metrics localize the break in one look —
  both ends will swear they're fine.
- Config-file consumers need a restart (or a reloader sidecar) after
  ConfigMap changes — the mounted file updated within a minute; the
  process's parsed config didn't. Second time this course has paged you
  for that (M3.9).
- Endpoint names in exporter configs are environment-specific — the
  pasted-from-another-cluster endpoint is this pattern's most common
  trigger, same genus as M2.19's copy-pasted resource block.
- `tls.insecure: true` is lab-grade; real pipelines pin certs exactly as
  M6.14 taught for images — telemetry is data too.

</details>
