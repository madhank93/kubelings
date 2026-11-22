---
kind: lesson
title: 'Deployments: let Kubernetes keep your pods alive'
description: |
  A bare pod dies and stays dead. Hand the job to a Deployment: it keeps the
  number of pods you asked for, replaces the ones that disappear, and scales
  with one command.
name: deployments-self-heal
slug: deployments-self-heal
createdAt: "2026-11-22"
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
      # A leftover bare pod labelled app=web would be adopted by the Deployment.
      kubectl -n "$NS" delete deployment web --ignore-not-found --wait=true
      kubectl -n "$NS" delete pod web --ignore-not-found --wait=true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get deployment web >/dev/null 2>&1; then
        echo "not yet: no Deployment named 'web' in namespace '$NS'"; exit 1
      fi
      image=$(kubectl -n "$NS" get deployment web -o jsonpath='{.spec.template.spec.containers[0].image}')
      if [ "$image" != "nginx:1.27-alpine" ]; then
        echo "not yet: Deployment 'web' runs image '$image', want nginx:1.27-alpine"; exit 1
      fi
      want=$(kubectl -n "$NS" get deployment web -o jsonpath='{.spec.replicas}')
      if [ "$want" != "3" ]; then
        echo "not yet: Deployment 'web' asks for $want replica(s), want 3"; exit 1
      fi
      ready=$(kubectl -n "$NS" get deployment web -o jsonpath='{.status.readyReplicas}')
      if [ "${ready:-0}" != "3" ]; then
        echo "not yet: ${ready:-0}/3 pods ready — give it a few seconds"; exit 1
      fi
      echo "PASS — three replicas, kept alive by the Deployment."
---
