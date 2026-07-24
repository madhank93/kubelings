---
kind: lesson
title: 'Kustomize: one base, many environments, zero drift'
description: |
  Staging and prod were "the same YAML, copy-pasted" — until someone hotfixed
  prod by hand and the copies disagreed. Rebuild the setup as a kustomize base
  plus a prod overlay with kubectl's built-in -k, and make the drift
  impossible to express.
name: kustomize-overlays
slug: kustomize-overlays
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
      kubectl -n "$NS" delete deploy api prod-api --ignore-not-found
      kubectl -n "$NS" delete svc api prod-api --ignore-not-found
      # The hand-drifted "prod" that someone kubectl-edited months ago:
      # 1 replica (a hotfix that never got reverted), no env label, stale image.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: api
      spec:
        replicas: 1
        selector:
          matchLabels: {app: api}
        template:
          metadata:
            labels: {app: api}
          spec:
            containers:
              - name: api
                image: nginx:1.25-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      YAML
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get deploy prod-api >/dev/null 2>&1; then
        echo "not yet: no 'prod-api' deployment — the prod overlay (namePrefix: prod-) hasn't been applied with kubectl apply -k"; exit 1
      fi
      env=$(kubectl -n "$NS" get deploy prod-api -o jsonpath='{.metadata.labels.env}' 2>/dev/null)
      if [ "$env" != "prod" ]; then
        echo "not yet: prod-api has no env=prod label — the overlay's common label isn't landing (labels:/commonLabels in the overlay kustomization)"; exit 1
      fi
      reps=$(kubectl -n "$NS" get deploy prod-api -o jsonpath='{.spec.replicas}')
      if [ "${reps:-0}" -lt 3 ]; then
        echo "not yet: prod-api runs $reps replica(s) — prod's replica patch (3) isn't applied"; exit 1
      fi
      img=$(kubectl -n "$NS" get deploy prod-api -o jsonpath='{.spec.template.spec.containers[0].image}')
      if [ "$img" != "nginx:1.27-alpine" ]; then
        echo "not yet: prod-api runs $img — the overlay should pin nginx:1.27-alpine via images:"; exit 1
      fi
      avail=$(kubectl -n "$NS" get deploy prod-api -o jsonpath='{.status.availableReplicas}')
      if [ "${avail:-0}" -lt 3 ]; then
        echo "not yet: prod-api is ${avail:-0}/3 — wait for the rollout"; exit 1
      fi
      if kubectl -n "$NS" get deploy api >/dev/null 2>&1; then
        echo "not yet: the old hand-drifted 'api' deployment is still there — delete it; the overlay-built prod-api replaces it"; exit 1
      fi
      echo "PASS — prod is now generated from base + overlay. Nobody can hand-edit what a build regenerates."
---
