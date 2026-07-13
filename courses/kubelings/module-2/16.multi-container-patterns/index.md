---
kind: lesson
title: 'Three pods, three broken patterns'
description: |
  Sidecar, ambassador, adapter — the three classic multi-container patterns,
  each deployed here with one bug: a log-forwarder tailing the wrong path, a
  proxy pointed at the wrong port, an adapter writing to the wrong place.
  Fix all three and prove each pattern actually does its job.
name: multi-container-patterns
slug: multi-container-patterns
createdAt: "2026-07-13"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 240
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      # Fresh pods each init: the fixes require pod recreation anyway.
      kubectl -n "$NS" delete pod orders-logs orders-checkout orders-metrics --ignore-not-found --wait=true
      kubectl apply -n "$NS" -f - <<'YAML'
      # ── 1. SIDECAR: app writes logs, forwarder ships them ──────────────
      apiVersion: v1
      kind: Pod
      metadata:
        name: orders-logs
        labels: {app: orders-logs, pattern: sidecar}
      spec:
        containers:
          - name: app
            image: busybox:1.36
            command: ["sh", "-c", "while true; do echo \"$(date -Iseconds) order shipped id=$RANDOM\" >> /var/log/app/app.log; sleep 2; done"]
            volumeMounts:
              - {name: logs, mountPath: /var/log/app}
            resources:
              requests: {cpu: 10m, memory: 16Mi}
              limits: {memory: 64Mi}
          - name: log-forwarder
            image: busybox:1.36
            # BUG: mounts the wrong volume — tails an empty decoy file on
            # `scratch` while the app writes into `logs`.
            command: ["sh", "-c", "touch /var/log/app/app.log; tail -F /var/log/app/app.log"]
            volumeMounts:
              - {name: scratch, mountPath: /var/log/app}
            resources:
              requests: {cpu: 10m, memory: 16Mi}
              limits: {memory: 64Mi}
        volumes:
          - name: logs
            emptyDir: {}
          - name: scratch
            emptyDir: {}
      ---
      # ── 2. AMBASSADOR: app talks to localhost, proxy reaches the service ──
      apiVersion: v1
      kind: Pod
      metadata:
        name: payments
        labels: {app: payments}
      spec:
        containers:
          - name: payments
            image: nginx:1.25-alpine
            ports: [{containerPort: 80}]
            resources:
              requests: {cpu: 10m, memory: 32Mi}
              limits: {memory: 128Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: payments
      spec:
        selector: {app: payments}
        ports: [{port: 80, targetPort: 80}]
      ---
      apiVersion: v1
      kind: Pod
      metadata:
        name: orders-checkout
        labels: {app: orders-checkout, pattern: ambassador}
      spec:
        containers:
          - name: app
            image: busybox:1.36
            command: ["sh", "-c", "while true; do sleep 30; done"]
            resources:
              requests: {cpu: 10m, memory: 16Mi}
              limits: {memory: 64Mi}
          - name: ambassador
            image: alpine/socat:1.8.0.0
            # BUG: payments serves on 80; nothing listens on 9999.
            args: ["TCP-LISTEN:8000,fork,reuseaddr", "TCP:payments:9999"]
            resources:
              requests: {cpu: 10m, memory: 16Mi}
              limits: {memory: 64Mi}
      ---
      # ── 3. ADAPTER: legacy metrics in, Prometheus format out ───────────
      apiVersion: v1
      kind: Pod
      metadata:
        name: orders-metrics
        labels: {app: orders-metrics, pattern: adapter}
      spec:
        containers:
          - name: app
            image: busybox:1.36
            command: ["sh", "-c", "i=0; while true; do i=$((i+1)); echo \"legacy|orders_processed|$i\" >> /shared/metrics.log; sleep 2; done"]
            volumeMounts:
              - {name: shared, mountPath: /shared}
            resources:
              requests: {cpu: 10m, memory: 16Mi}
              limits: {memory: 64Mi}
          - name: adapter
            # BUG: converts correctly but prints to stdout — the scraper
            # reads /shared/metrics.prom, which never appears.
            image: busybox:1.36
            command: ["sh", "-c", "while true; do if [ -f /shared/metrics.log ]; then awk -F'|' '{print \"orders_processed_total \" $3}' /shared/metrics.log | tail -1; fi; sleep 2; done"]
            volumeMounts:
              - {name: shared, mountPath: /shared}
            resources:
              requests: {cpu: 10m, memory: 16Mi}
              limits: {memory: 64Mi}
        volumes:
          - name: shared
            emptyDir: {}
      YAML
      kubectl -n "$NS" wait --for=condition=Ready pod/orders-logs pod/orders-checkout pod/orders-metrics pod/payments --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      for p in orders-logs orders-checkout orders-metrics; do
        phase=$(kubectl -n "$NS" get pod "$p" -o jsonpath='{.status.phase}' 2>/dev/null)
        if [ "$phase" != "Running" ]; then
          echo "not yet: pod $p is ${phase:-missing} — all three pods must be Running"; exit 1
        fi
      done
      # 1. sidecar: forwarder must actually be shipping the app's lines
      if ! kubectl -n "$NS" logs orders-logs -c log-forwarder --tail=5 2>/dev/null | grep -q "order shipped"; then
        echo "not yet: log-forwarder in orders-logs ships nothing — is it tailing the path the app writes to?"; exit 1
      fi
      # 2. ambassador: app must reach payments through localhost:8000
      if ! kubectl -n "$NS" exec orders-checkout -c app -- wget -qO- -T 5 http://127.0.0.1:8000 >/dev/null 2>&1; then
        echo "not yet: orders-checkout app cannot reach payments via localhost:8000 — where does the ambassador forward to?"; exit 1
      fi
      # 3. adapter: converted metrics must land in the shared file
      if ! kubectl -n "$NS" exec orders-metrics -c app -- cat /shared/metrics.prom 2>/dev/null | grep -Eq '^orders_processed_total [0-9]+'; then
        echo "not yet: /shared/metrics.prom is missing or empty in orders-metrics — the adapter converts fine, but where does its output go?"; exit 1
      fi
      echo "PASS — sidecar ships, ambassador proxies, adapter translates. All three patterns doing their one job."
---
