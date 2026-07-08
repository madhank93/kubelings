---
kind: lesson
title: 'Final boss: three faults, no hints'
description: |
  A checkout stack with three independent faults stacked on top of each other —
  a selector mismatch, a bad probe, and an impossible resource request. No hints,
  no fault list. Diagnose each from first principles and ship the fix. This is
  the exam.
name: final-boss
slug: final-boss
createdAt: "2026-07-07"
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
      # FAULT A: api Service selects the wrong label -> empty endpoints (M1).
      # FAULT B: web liveness probe wrong port -> restart storm (M2).
      # FAULT C: worker requests 500Gi memory -> unschedulable (M8).
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata: {name: api}
      spec:
        replicas: 2
        selector: {matchLabels: {app: api}}
        template:
          metadata: {labels: {app: api}}
          spec:
            containers:
              - name: api
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                resources: {requests: {cpu: 10m, memory: 32Mi}, limits: {memory: 128Mi}}
      ---
      apiVersion: v1
      kind: Service
      metadata: {name: api}
      spec:
        selector: {app: api-server}   # FAULT A: pods are app=api
        ports: [{port: 80, targetPort: 80}]
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata: {name: web}
      spec:
        replicas: 2
        selector: {matchLabels: {app: web}}
        template:
          metadata: {labels: {app: web}}
          spec:
            containers:
              - name: web
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                livenessProbe:          # FAULT B: app on 80, probe on 8080
                  httpGet: {path: /, port: 8080}
                  initialDelaySeconds: 5
                  periodSeconds: 5
                  failureThreshold: 2
                resources: {requests: {cpu: 10m, memory: 32Mi}, limits: {memory: 128Mi}}
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata: {name: worker}
      spec:
        replicas: 1
        selector: {matchLabels: {app: worker}}
        template:
          metadata: {labels: {app: worker}}
          spec:
            containers:
              - name: worker
                image: busybox:1.36
                command: ["sh", "-c", "while true; do sleep 10; done"]
                resources: {requests: {cpu: 10m, memory: 500Gi}, limits: {memory: 500Gi}}  # FAULT C
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # A: api Service has 2 endpoints and serves.
      addrs=$(kubectl -n "$NS" get endpoints api -o jsonpath='{range .subsets[*].addresses[*]}{.ip}{"\n"}{end}' 2>/dev/null | grep -c . || true)
      if [ "${addrs:-0}" -lt 2 ]; then
        echo "FAULT A not fixed: api Service has ${addrs:-0} endpoints (expected 2)"; exit 1
      fi
      if ! kubectl -n "$NS" run boss-check --rm -i --restart=Never --image=busybox:1.36 \
          --timeout=60s -- wget -q -O- --timeout=5 "http://api.$NS.svc/" 2>/dev/null | grep -qi nginx; then
        echo "FAULT A not fixed: http://api.$NS.svc/ still fails"; exit 1
      fi
      # B: web available and not restart-storming.
      wavail=$(kubectl -n "$NS" get deploy web -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ "${wavail:-0}" -lt 2 ]; then
        echo "FAULT B not fixed: web is ${wavail:-0}/2 Available"; exit 1
      fi
      newest=$(kubectl -n "$NS" get pods -l app=web --sort-by=.metadata.creationTimestamp \
        -o jsonpath='{.items[-1:].status.containerStatuses[0].restartCount}' 2>/dev/null)
      if [ "${newest:-0}" -gt 1 ]; then
        echo "FAULT B not fixed: newest web pod restart count is $newest (probe still wrong?)"; exit 1
      fi
      # C: worker scheduled and available.
      cavail=$(kubectl -n "$NS" get deploy worker -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ "${cavail:-0}" -lt 1 ]; then
        echo "FAULT C not fixed: worker still not Available (resource request too large?)"; exit 1
      fi
      echo "PASS — all three faults down. You diagnosed empty endpoints, a killer probe, and an impossible request with no hints. You know this platform."
---
