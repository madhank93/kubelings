---
kind: lesson
title: 'Pattern drill: evicted — the disk you forgot to budget'
description: |
  Pods keep dying with a status you haven't seen before: Evicted. No crash, no
  OOM, exit code nowhere — the kubelet itself killed them for eating too much
  disk. Ephemeral storage is the resource nobody puts in their manifests, and
  eviction is how you find out.
name: pattern-disk-pressure
slug: pattern-disk-pressure
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
      kubectl -n "$NS" delete deploy report-builder --ignore-not-found
      kubectl -n "$NS" delete pod --field-selector=status.phase=Failed --ignore-not-found >/dev/null 2>&1 || true
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: report-builder
      spec:
        replicas: 2
        selector:
          matchLabels: {app: report-builder}
        template:
          metadata:
            labels: {app: report-builder}
          spec:
            containers:
              - name: builder
                image: busybox:1.36
                # Cache warmup: materializes ~200Mi of report fragments on
                # local disk, then serves from them.
                command:
                  - sh
                  - -c
                  - |
                    mkdir -p /scratch
                    i=0
                    while [ $i -lt 40 ]; do
                      dd if=/dev/zero of=/scratch/chunk-$i bs=1048576 count=5 2>/dev/null
                      i=$((i+1))
                      sleep 1
                    done
                    echo "warmup complete"
                    while true; do sleep 3600; done
                resources:
                  requests:
                    cpu: 10m
                    memory: 32Mi
                    ephemeral-storage: 20Mi
                  limits:
                    # BUG: warmup needs ~200Mi; the kubelet evicts at 64Mi.
                    ephemeral-storage: 64Mi
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      lim=$(kubectl -n "$NS" get deploy report-builder -o jsonpath='{.spec.template.spec.containers[0].resources.limits.ephemeral-storage}' 2>/dev/null)
      if [ -z "$lim" ]; then
        echo "not yet: the builder container has no ephemeral-storage limit at all — unbounded disk is how one pod takes out a node"; exit 1
      fi
      case "$lim" in
        *Gi) mib=$(( ${lim%Gi} * 1024 )) ;;
        *Mi) mib=${lim%Mi} ;;
        *)   mib=0 ;;
      esac
      if [ "$mib" -lt 250 ]; then
        echo "not yet: ephemeral-storage limit is $lim but the warmup writes ~200Mi — budget the real footprint (with headroom)"; exit 1
      fi
      desired=$(kubectl -n "$NS" get deploy report-builder -o jsonpath='{.spec.replicas}')
      avail=$(kubectl -n "$NS" get deploy report-builder -o jsonpath='{.status.availableReplicas}')
      if [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: report-builder is ${avail:-0}/$desired — wait for the warmup to survive past the old kill line"; exit 1
      fi
      corpses=$(kubectl -n "$NS" get pods --field-selector=status.phase=Failed --no-headers 2>/dev/null | wc -l | tr -d ' ')
      if [ "$corpses" -gt 0 ]; then
        echo "not yet: $corpses Evicted pod(s) still lying around — controllers don't clean those up; delete the Failed pods"; exit 1
      fi
      echo "PASS — disk budgeted like memory, warmup survives, corpses buried. Ephemeral storage is a real resource; treat it like one."
---
