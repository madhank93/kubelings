---
kind: lesson
title: 'One disk, two nodes: the access-mode trap'
description: |
  The wiki scaled to two replicas for "high availability" — and the second pod
  has been Pending ever since. Both want the same ReadWriteOnce volume, and
  anti-affinity pushes them to different nodes. Learn what access modes really
  promise and why RWO + spread = deadlock.
name: access-modes
slug: access-modes
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
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: PersistentVolumeClaim
      metadata:
        name: wiki-data
      spec:
        accessModes: [ReadWriteOnce]
        storageClassName: standard
        resources:
          requests: {storage: 1Gi}
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: wiki
      spec:
        replicas: 2
        selector:
          matchLabels: {app: wiki}
        template:
          metadata:
            labels: {app: wiki}
          spec:
            # BUG(combo): required anti-affinity forces replicas onto different
            # nodes, but they share one RWO (node-local) volume. Pod 2 can never run.
            affinity:
              podAntiAffinity:
                requiredDuringSchedulingIgnoredDuringExecution:
                  - labelSelector:
                      matchLabels: {app: wiki}
                    topologyKey: kubernetes.io/hostname
            containers:
              - name: wiki
                image: busybox:1.36
                command: ["sh", "-c", "echo wiki up; while true; do sleep 5; done"]
                volumeMounts:
                  - {name: data, mountPath: /var/wiki}
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
            volumes:
              - name: data
                persistentVolumeClaim:
                  claimName: wiki-data
      YAML
      sleep 10 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      desired=$(kubectl -n "$NS" get deploy wiki -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy wiki -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: wiki not fully Available (${avail:-0}/${desired:-?}) — why can't the Pending pod schedule?"; exit 1
      fi
      pending=$(kubectl -n "$NS" get pods -l app=wiki --field-selector=status.phase=Pending -o name 2>/dev/null | grep -c . || true)
      if [ "${pending:-0}" -gt 0 ]; then
        echo "not yet: $pending wiki pod(s) still Pending"; exit 1
      fi
      echo "PASS — placement and storage agree. RWO means one NODE at a time; design around it, don't fight it."
---
