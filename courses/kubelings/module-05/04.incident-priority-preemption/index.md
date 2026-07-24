---
kind: lesson
title: 'Incident replay — the priority that ate production (Grafana Labs)'
description: |
  Replay of Grafana Labs' cited outage: introducing PriorityClasses with the
  wrong defaults caused Kubernetes to PREEMPT — evict — production pods to make
  room. Set the priority tiers right so the important thing wins the next
  resource fight.
name: incident-priority-preemption
slug: incident-priority-preemption
source: https://grafana.com/blog/2019/07/24/how-a-production-outage-was-caused-using-kubernetes-pod-priorities/
createdAt: "2026-07-07"
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
      kubectl delete priorityclass tier-critical tier-batch --ignore-not-found
      kubectl apply -f - <<'YAML'
      apiVersion: scheduling.k8s.io/v1
      kind: PriorityClass
      metadata:
        name: tier-batch
      # BUG: batch got the HIGH number in the copy-paste. Under pressure, the
      # scheduler will evict checkout pods to make room for reindexing jobs.
      value: 100000
      preemptionPolicy: PreemptLowerPriority
      description: "batch / best-effort work"
      ---
      apiVersion: scheduling.k8s.io/v1
      kind: PriorityClass
      metadata:
        name: tier-critical
      value: 1000
      preemptionPolicy: PreemptLowerPriority
      description: "revenue-critical services"
      YAML
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: checkout
      spec:
        replicas: 2
        selector:
          matchLabels: {app: checkout}
        template:
          metadata:
            labels: {app: checkout}
          spec:
            priorityClassName: tier-critical
            containers:
              - name: checkout
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: reindex
      spec:
        replicas: 2
        selector:
          matchLabels: {app: reindex}
        template:
          metadata:
            labels: {app: reindex}
          spec:
            priorityClassName: tier-batch
            containers:
              - name: reindex
                image: busybox:1.36
                command: ["sh", "-c", "while true; do sleep 10; done"]
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/checkout --timeout=180s
      kubectl -n "$NS" rollout status deploy/reindex --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      crit=$(kubectl get priorityclass tier-critical -o jsonpath='{.value}' 2>/dev/null)
      batch=$(kubectl get priorityclass tier-batch -o jsonpath='{.value}' 2>/dev/null)
      if [ -z "$crit" ] || [ -z "$batch" ]; then
        echo "not yet: keep both PriorityClasses — fix their values, don't delete the tiers"; exit 1
      fi
      if [ "$crit" -le "$batch" ]; then
        echo "not yet: tier-critical ($crit) must outrank tier-batch ($batch) — right now batch preempts checkout"; exit 1
      fi
      # Batch should not be able to preempt anything: the polite setting.
      pol=$(kubectl get priorityclass tier-batch -o jsonpath='{.preemptionPolicy}' 2>/dev/null)
      if [ "$pol" != "Never" ]; then
        echo "not yet: tier-batch preemptionPolicy is '$pol' — batch work should wait, not evict (set Never)"; exit 1
      fi
      for d in checkout reindex; do
        avail=$(kubectl -n "$NS" get deploy "$d" -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
        if [ "${avail:-0}" -lt 1 ]; then echo "not yet: $d not Available"; exit 1; fi
      done
      echo "PASS — tiers ordered, batch preemption disarmed. The next resource fight already has a winner: production."
---
