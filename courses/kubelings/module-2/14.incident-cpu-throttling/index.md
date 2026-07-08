---
kind: lesson
title: 'CPU throttling: the latency tax nobody sees'
description: |
  The service is "healthy" — CPU usage looks modest, no restarts, no errors —
  and p95 latency is garbage. The kernel is pausing your process dozens of
  times a second to enforce a CPU limit. Read the throttling counters the
  kernel keeps inside every container, then fix the limit like Omio and
  Buffer had to.
name: incident-cpu-throttling
slug: incident-cpu-throttling
source: https://medium.com/omio-engineering/cpu-limits-and-aggressive-throttling-in-kubernetes-c5b20bd8a718
createdAt: "2026-07-08"
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
      kubectl -n "$NS" delete deploy pricing --ignore-not-found
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: pricing
      spec:
        replicas: 1
        selector:
          matchLabels: {app: pricing}
        template:
          metadata:
            labels: {app: pricing}
          spec:
            containers:
              - name: pricing
                image: busybox:1.36
                # Simulates request bursts: ~200ms of computation, then idle.
                # Average CPU is low; burst demand is a full core.
                command:
                  - sh
                  - -c
                  - |
                    while true; do
                      i=0; while [ $i -lt 200000 ]; do i=$((i+1)); done
                      sleep 0.8
                    done
                resources:
                  requests: {cpu: 25m, memory: 16Mi}
                  # BUG: 50m = 5ms of CPU per 100ms window. Every burst gets
                  # paused ~40 times. Average usage still looks tiny.
                  limits: {cpu: 50m, memory: 64Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/pricing --timeout=120s
      # Let it accumulate some throttling history before the student looks.
      sleep 10 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      req=$(kubectl -n "$NS" get deploy pricing -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}' 2>/dev/null)
      if [ -z "$req" ]; then
        echo "not yet: pricing has no CPU request — requests are the scheduler's truth; never drop them"; exit 1
      fi
      lim=$(kubectl -n "$NS" get deploy pricing -o jsonpath='{.spec.template.spec.containers[0].resources.limits.cpu}' 2>/dev/null)
      if [ -n "$lim" ]; then
        # Accept a generous limit (>= 500m) — the point is burst headroom.
        case "$lim" in
          *m) n=${lim%m} ;;
          *)  n=$(( ${lim%%.*} * 1000 )) ;;
        esac
        if [ "${n:-0}" -lt 500 ]; then
          echo "not yet: CPU limit is $lim — bursts still hit the CFS wall. Give real headroom (>=500m) or remove the CPU limit entirely"; exit 1
        fi
      fi
      avail=$(kubectl -n "$NS" get deploy pricing -o jsonpath='{.status.availableReplicas}')
      if [ "${avail:-0}" -lt 1 ]; then
        echo "not yet: pricing isn't running — wait for the rollout"; exit 1
      fi
      echo "PASS — requests for scheduling truth, burst headroom for latency. Check cpu.stat again: nr_throttled should stop climbing."
---
