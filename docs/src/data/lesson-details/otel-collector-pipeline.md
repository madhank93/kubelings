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
