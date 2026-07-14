---
kind: lesson
title: 'KEDA: the ScaledObject that never scaled'
description: |
  Nightly report traffic needs 3 replicas during business hours; HPA can't
  scale on "what time it is" — KEDA can. But this ScaledObject sits
  Ready=False and the deployment never leaves 1 replica: the cron trigger's
  schedule doesn't parse. Fix the trigger, watch KEDA drive the HPA it
  manages for you.
name: keda-autoscaling
slug: keda-autoscaling
createdAt: "2026-07-14"
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
      # KEDA, pinned official release manifest.
      kubectl apply --server-side --force-conflicts -f \
        https://github.com/kedacore/keda/releases/download/v2.20.1/keda-2.20.1.yaml
      kubectl -n keda rollout status deploy/keda-operator --timeout=240s
      kubectl -n keda rollout status deploy/keda-metrics-apiserver --timeout=240s
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: report-web
        labels: {app: report-web}
      spec:
        replicas: 1
        selector:
          matchLabels: {app: report-web}
        template:
          metadata:
            labels: {app: report-web}
          spec:
            containers:
              - name: report-web
                image: busybox:1.36
                command: ["sh", "-c", "while true; do sleep 30; done"]
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      ---
      apiVersion: keda.sh/v1alpha1
      kind: ScaledObject
      metadata:
        name: report-web
      spec:
        scaleTargetRef:
          name: report-web
        minReplicaCount: 1
        maxReplicaCount: 3
        triggers:
          - type: cron
            metadata:
              timezone: Etc/UTC
              # BUG: there is no hour 25 — the schedule never parses,
              # the trigger never activates, nothing ever scales.
              start: 0 25 * * *
              end: 59 23 * * *
              desiredReplicas: "3"
      YAML
      kubectl -n "$NS" rollout status deploy/report-web --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    timeout_seconds: 300
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get scaledobject report-web >/dev/null 2>&1; then
        echo "not yet: ScaledObject report-web is gone — fix the trigger, don't delete it"; exit 1
      fi
      ready=$(kubectl -n "$NS" get scaledobject report-web -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)
      if [ "$ready" != "True" ]; then
        msg=$(kubectl -n "$NS" get scaledobject report-web -o jsonpath='{.status.conditions[?(@.type=="Ready")].message}' 2>/dev/null)
        echo "not yet: ScaledObject not Ready — ${msg:-check the trigger}; is that cron schedule a real time?"; exit 1
      fi
      if ! kubectl -n "$NS" get hpa keda-hpa-report-web >/dev/null 2>&1; then
        echo "not yet: KEDA hasn't created its HPA (keda-hpa-report-web) — the ScaledObject must be Ready first"; exit 1
      fi
      reps=$(kubectl -n "$NS" get deploy report-web -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
      if [ "${reps:-0}" -lt 3 ]; then
        echo "not yet: report-web has ${reps:-0}/3 replicas — the cron window should be active now (scaling takes ~30s after the fix; if it's 23:59-00:00 UTC you found the window gap, wait a minute)"; exit 1
      fi
      echo "PASS — trigger parses, KEDA's HPA is driving, 3 replicas in the window. Time-based scaling, no metrics stack required."
---
