---
kind: lesson
title: 'Tags lie, digests don''t: pin the supply chain'
description: |
  A tag like nginx:1.27-alpine is a mutable pointer — whoever controls the
  registry controls what it points at, and yesterday's audited image may not
  be tomorrow's. Find the digest of what checkout is actually running and pin
  the deployment to it, making the image content-addressed and unswappable.
name: image-digests
slug: image-digests
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
      kubectl -n "$NS" delete deploy checkout --ignore-not-found
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
            containers:
              - name: checkout
                image: nginx:1.27-alpine
                imagePullPolicy: IfNotPresent
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/checkout --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      img=$(kubectl -n "$NS" get deploy checkout -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null)
      case "$img" in
        *@sha256:*) : ;;
        *)
          echo "not yet: checkout still runs '$img' — a mutable tag. Pin it to the digest of what's running (hint: the pod's .status.containerStatuses imageID)"; exit 1 ;;
      esac
      digest="${img##*@}"
      if ! echo "$digest" | grep -Eq '^sha256:[0-9a-f]{64}$'; then
        echo "not yet: '$digest' isn't a valid sha256 digest — copy it exactly from the running pod's imageID"; exit 1
      fi
      running=$(kubectl -n "$NS" get pods -l app=checkout -o jsonpath='{.items[0].status.containerStatuses[0].imageID}' 2>/dev/null)
      case "$running" in
        *"$digest"*) : ;;
        *)
          echo "not yet: the deployment pins $digest but the running pods report a different imageID — pin the digest you verified, not one you invented"; exit 1 ;;
      esac
      desired=$(kubectl -n "$NS" get deploy checkout -o jsonpath='{.spec.replicas}')
      avail=$(kubectl -n "$NS" get deploy checkout -o jsonpath='{.status.availableReplicas}')
      if [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: checkout is ${avail:-0}/$desired after the pin — wait for the rollout (or check the digest resolves)"; exit 1
      fi
      echo "PASS — checkout now runs content, not a pointer. This exact byte-for-byte image, or nothing."
---
