---
kind: lesson
title: 'SLOs: the burn-rate alert that never fired'
description: |
  An SLO burn-rate alert has been silently broken for months: its
  PrometheusRule references a metric that doesn't exist, and the "error
  ratio" counts 404s as server errors. Fix both bugs and prove the burn
  rate evaluates against live traffic.
name: slo-errorbudget
slug: slo-errorbudget
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
      # prometheus-operator, pinned bundle (CRDs + operator, kube-system-free).
      kubectl apply --server-side --force-conflicts -f \
        https://github.com/prometheus-operator/prometheus-operator/releases/download/v0.92.1/bundle.yaml
      kubectl -n default rollout status deploy/prometheus-operator --timeout=180s
      # Synthetic service: real HTTP metrics (http_requests_total by code)...
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: checkout-api
        labels: {app: checkout-api}
      spec:
        replicas: 1
        selector:
          matchLabels: {app: checkout-api}
        template:
          metadata:
            labels: {app: checkout-api}
          spec:
            containers:
              - name: app
                image: quay.io/brancz/prometheus-example-app:v0.5.0
                ports: [{name: http, containerPort: 8080}]
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 64Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: checkout-api
        labels: {app: checkout-api}
      spec:
        selector: {app: checkout-api}
        ports: [{name: http, port: 8080, targetPort: 8080}]
      ---
      # ...and traffic with a healthy dose of 404s (a scanner bot's greatest hits).
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: traffic-gen
        labels: {app: traffic-gen}
      spec:
        replicas: 1
        selector:
          matchLabels: {app: traffic-gen}
        template:
          metadata:
            labels: {app: traffic-gen}
          spec:
            containers:
              - name: gen
                image: busybox:1.36
                command: ["sh", "-c", "while true; do wget -qO- http://checkout-api:8080/ >/dev/null 2>&1; wget -qO- http://checkout-api:8080/ >/dev/null 2>&1; wget -qO- http://checkout-api:8080/err >/dev/null 2>&1; sleep 2; done"]
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      YAML
      # Prometheus itself: SA + minimal RBAC + a 1-replica server.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: ServiceAccount
      metadata:
        name: prometheus
      ---
      apiVersion: rbac.authorization.k8s.io/v1
      kind: ClusterRole
      metadata:
        name: kubelings-prometheus
      rules:
        - apiGroups: [""]
          resources: [services, endpoints, pods]
          verbs: [get, list, watch]
        - nonResourceURLs: ["/metrics"]
          verbs: [get]
      ---
      apiVersion: rbac.authorization.k8s.io/v1
      kind: ClusterRoleBinding
      metadata:
        name: kubelings-prometheus
      roleRef:
        apiGroup: rbac.authorization.k8s.io
        kind: ClusterRole
        name: kubelings-prometheus
      subjects:
        - kind: ServiceAccount
          name: prometheus
          namespace: kubelings
      ---
      apiVersion: monitoring.coreos.com/v1
      kind: Prometheus
      metadata:
        name: kubelings
      spec:
        replicas: 1
        serviceAccountName: prometheus
        serviceMonitorSelector:
          matchLabels: {team: kubelings}
        ruleSelector:
          matchLabels: {team: kubelings}
        resources:
          requests: {cpu: 100m, memory: 256Mi}
          limits: {memory: 512Mi}
      ---
      apiVersion: monitoring.coreos.com/v1
      kind: ServiceMonitor
      metadata:
        name: checkout-api
        labels: {team: kubelings}
      spec:
        selector:
          matchLabels: {app: checkout-api}
        endpoints:
          - port: http
            interval: 15s
      ---
      # The broken SLO rules. Bug 1: metric name that doesn't exist.
      # Bug 2: 404s counted as error budget burn.
      apiVersion: monitoring.coreos.com/v1
      kind: PrometheusRule
      metadata:
        name: checkout-slo
        labels: {team: kubelings}
      spec:
        groups:
          - name: checkout-slo
            rules:
              - record: checkout:request_rate5m
                expr: sum(rate(http_request_total[5m]))
              - record: checkout:error_ratio5m
                expr: |
                  sum(rate(http_requests_total{code=~"4..|5.."}[5m]))
                  /
                  sum(rate(http_requests_total[5m]))
              - alert: CheckoutErrorBudgetBurn
                expr: checkout:error_ratio5m > 0.02
                for: 5m
                labels: {severity: page}
                annotations:
                  summary: "checkout is burning error budget fast"
      YAML
      kubectl -n "$NS" rollout status deploy/checkout-api --timeout=120s
      kubectl -n "$NS" wait --for=condition=Available deploy/traffic-gen --timeout=120s
      # Operator spins up the statefulset asynchronously — wait for the pod.
      for i in $(seq 1 60); do
        kubectl -n "$NS" get pod prometheus-kubelings-0 >/dev/null 2>&1 && break
        sleep 2
      done
      kubectl -n "$NS" wait --for=condition=Ready pod/prometheus-kubelings-0 --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    timeout_seconds: 300
    run: |
      NS=kubelings
      rules=$(kubectl -n "$NS" get prometheusrule checkout-slo -o yaml 2>/dev/null)
      if [ -z "$rules" ]; then
        echo "not yet: PrometheusRule checkout-slo is gone — fix the rules, don't delete them"; exit 1
      fi
      if echo "$rules" | grep -q "http_request_total"; then
        echo "not yet: the request-rate rule still references http_request_total — that metric does not exist (check what checkout-api actually exports)"; exit 1
      fi
      if echo "$rules" | grep -q '4\.\.'; then
        echo "not yet: the error ratio still counts 4xx as errors — a 404 is the client's problem, not your error budget's"; exit 1
      fi
      # The recording rule must emit a sample against live traffic — zero
      # errors must still produce the number 0, not an empty vector.
      out=$(kubectl get --raw "/api/v1/namespaces/$NS/services/prometheus-operated:9090/proxy/api/v1/query?query=checkout:error_ratio5m" 2>/dev/null)
      case "$out" in
        *'"status":"success"'*'"value"'*) : ;;
        *) echo "not yet: checkout:error_ratio5m returns no sample — with zero 5xx, a plain sum(rate(...{code=~\"5..\"})) is an EMPTY vector, and empty/total is empty. Make the numerator fall back to 0 (or vector(0)); rules reload takes ~1 min"; exit 1 ;;
      esac
      val=$(printf '%s' "$out" | sed 's/.*"value":\[[^,]*,"\([^"]*\)".*/\1/')
      ok=$(awk -v v="$val" 'BEGIN{print (v+0 < 0.02) ? "yes" : "no"}')
      if [ "$ok" != "yes" ]; then
        echo "not yet: burn rate is $val — with only 200s and 404s in the traffic, a correct 5xx-only ratio reads 0. Something still counts client errors"; exit 1
      fi
      echo "PASS — rules reference real metrics, 404s no longer burn budget, and the ratio emits an honest 0 instead of an empty vector."
---
