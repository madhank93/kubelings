---
kind: lesson
title: 'OTel pipeline: traces into the void'
description: |
  An OpenTelemetry Collector receives traces from every service and exports
  them to Jaeger — except Jaeger has been empty for a week. The collector's
  exporter points at a service name that doesn't exist, and it retries into
  the void. Read the pipeline config, fix the endpoint, watch traces land.
name: otel-collector-pipeline
slug: otel-collector-pipeline
createdAt: "2026-07-13"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 300
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      kubectl apply -n "$NS" -f - <<'YAML'
      # Tracing backend: Jaeger all-in-one (OTLP ingest built in).
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: tracing-backend
        labels: {app: tracing-backend}
      spec:
        replicas: 1
        selector:
          matchLabels: {app: tracing-backend}
        template:
          metadata:
            labels: {app: tracing-backend}
          spec:
            containers:
              - name: jaeger
                image: jaegertracing/all-in-one:1.75.0
                ports:
                  - {name: otlp-grpc, containerPort: 4317}
                  - {name: query, containerPort: 16686}
                resources:
                  requests: {cpu: 50m, memory: 128Mi}
                  limits: {memory: 512Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: tracing-backend
        labels: {app: tracing-backend}
      spec:
        selector: {app: tracing-backend}
        ports:
          - {name: otlp-grpc, port: 4317, targetPort: 4317}
          - {name: query, port: 16686, targetPort: 16686}
      ---
      # The collector's pipeline config. BUG: exporter endpoint names a
      # service that does not exist in this cluster.
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: otel-collector-config
      data:
        config.yaml: |
          receivers:
            otlp:
              protocols:
                grpc:
                  endpoint: 0.0.0.0:4317
          processors:
            batch: {}
          exporters:
            otlp/jaeger:
              endpoint: jaeger-collector.observability:4317
              tls:
                insecure: true
          service:
            pipelines:
              traces:
                receivers: [otlp]
                processors: [batch]
                exporters: [otlp/jaeger]
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: otel-collector
        labels: {app: otel-collector}
      spec:
        replicas: 1
        selector:
          matchLabels: {app: otel-collector}
        template:
          metadata:
            labels: {app: otel-collector}
          spec:
            containers:
              - name: collector
                image: otel/opentelemetry-collector-contrib:0.156.0
                args: ["--config=/etc/otelcol/config.yaml"]
                ports:
                  - {name: otlp-grpc, containerPort: 4317}
                volumeMounts:
                  - {name: config, mountPath: /etc/otelcol}
                resources:
                  requests: {cpu: 50m, memory: 128Mi}
                  limits: {memory: 256Mi}
            volumes:
              - name: config
                configMap: {name: otel-collector-config, items: [{key: config.yaml, path: config.yaml}]}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: otel-collector
        labels: {app: otel-collector}
      spec:
        selector: {app: otel-collector}
        ports:
          - {name: otlp-grpc, port: 4317, targetPort: 4317}
      ---
      # The instrumented "app": a steady stream of spans into the collector.
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: span-source
        labels: {app: span-source}
      spec:
        replicas: 1
        selector:
          matchLabels: {app: span-source}
        template:
          metadata:
            labels: {app: span-source}
          spec:
            containers:
              - name: telemetrygen
                image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.156.0
                args:
                  - traces
                  - --otlp-endpoint=otel-collector:4317
                  - --otlp-insecure
                  - --rate=2
                  - --duration=24h
                  - --service=checkout
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 64Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/tracing-backend --timeout=180s
      kubectl -n "$NS" rollout status deploy/otel-collector --timeout=180s
      kubectl -n "$NS" rollout status deploy/span-source --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    timeout_seconds: 300
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" rollout status deploy/otel-collector --timeout=60s >/dev/null 2>&1; then
        echo "not yet: otel-collector deployment not healthy"; exit 1
      fi
      ep=$(kubectl -n "$NS" get configmap otel-collector-config -o jsonpath='{.data.config\.yaml}' 2>/dev/null | grep -A1 'otlp/jaeger' | grep 'endpoint:' | awk '{print $2}')
      case "$ep" in
        tracing-backend:4317|tracing-backend.kubelings*:4317) : ;;
        *) echo "not yet: exporter endpoint is '$ep' — what Service actually fronts Jaeger in this namespace?"; exit 1 ;;
      esac
      # Did the fixed config actually reach the running pod? (ConfigMap edits
      # don't restart deployments.)
      if kubectl -n "$NS" logs deploy/otel-collector --tail=30 2>/dev/null | grep -qi "connection error\|transient failure\|failed to resolve\|Exporting failed"; then
        echo "not yet: the running collector still logs export failures — did you restart it after fixing the ConfigMap? (kubectl rollout restart)"; exit 1
      fi
      # Traces must have landed: ask Jaeger's query API for known services.
      svcs=$(kubectl get --raw "/api/v1/namespaces/$NS/services/tracing-backend:16686/proxy/api/services" 2>/dev/null)
      case "$svcs" in
        *checkout*) : ;;
        *) echo "not yet: Jaeger has no 'checkout' service — spans aren't arriving; check collector logs end to end"; exit 1 ;;
      esac
      echo "PASS — pipeline flows: app -> collector -> Jaeger, and the checkout service's traces are queryable."
---
