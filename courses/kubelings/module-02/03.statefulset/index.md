---
kind: lesson
title: StatefulSet with Stable Pod Identity + Headless Service
description: |
  A stateful app needs stable, predictable pod names and per-pod DNS. Build a
  StatefulSet fronted by a headless Service so each replica gets a durable
  network identity, then confirm all replicas are Ready.
name: statefulset
slug: statefulset
createdAt: "2026-06-30"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 180
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # Headless Service: clusterIP must be None.
      kubectl -n "$NS" get svc web >/dev/null 2>&1 || { echo "not yet: no Service 'web'"; exit 1; }
      cip=$(kubectl -n "$NS" get svc web -o jsonpath='{.spec.clusterIP}')
      [ "$cip" = "None" ] || { echo "not yet: Service 'web' is not headless (clusterIP=$cip, want None)"; exit 1; }
      # StatefulSet must be fully ready.
      kubectl -n "$NS" get statefulset web >/dev/null 2>&1 || { echo "not yet: no StatefulSet 'web'"; exit 1; }
      desired=$(kubectl -n "$NS" get sts web -o jsonpath='{.spec.replicas}')
      ready=$(kubectl -n "$NS" get sts web -o jsonpath='{.status.readyReplicas}')
      if [ "${desired:-0}" -lt 2 ]; then echo "not yet: want at least 2 replicas (got ${desired:-0})"; exit 1; fi
      if [ "${ready:-0}" -ne "${desired:-0}" ]; then echo "not yet: $ready/$desired replicas Ready"; exit 1; fi
      # serviceName must wire the STS to the headless Service.
      svcname=$(kubectl -n "$NS" get sts web -o jsonpath='{.spec.serviceName}')
      [ "$svcname" = "web" ] || { echo "not yet: StatefulSet.spec.serviceName must be 'web' (got '$svcname')"; exit 1; }
      echo "PASS — StatefulSet web has $ready stable replicas behind headless Service web."
---
