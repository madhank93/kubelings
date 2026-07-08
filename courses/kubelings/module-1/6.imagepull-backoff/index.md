---
kind: lesson
title: 'ImagePullBackOff: the tag that never existed'
description: |
  A hotfix deploy is stuck — pods sit in ImagePullBackOff and the rollout never
  completes. The registry is fine. The image is fine. The tag is not. Learn to
  read pull errors and roll the Deployment back to health.
name: imagepull-backoff
slug: imagepull-backoff
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
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: frontend
      spec:
        replicas: 2
        selector:
          matchLabels: {app: frontend}
        template:
          metadata:
            labels: {app: frontend}
          spec:
            containers:
              - name: frontend
                # BUG: this tag does not exist — someone fat-fingered the hotfix version.
                image: nginx:1.27.9999-alpine
                ports: [{containerPort: 80}]
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      desired=$(kubectl -n "$NS" get deploy frontend -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy frontend -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: frontend not Available (${avail:-0}/${desired:-?}) — check the image events"; exit 1
      fi
      bad=$(kubectl -n "$NS" get pods -l app=frontend \
        -o jsonpath='{range .items[*]}{.status.containerStatuses[*].state.waiting.reason}{"\n"}{end}' 2>/dev/null \
        | grep -cE 'ImagePull|ErrImage' || true)
      if [ "${bad:-0}" -gt 0 ]; then
        echo "not yet: some frontend pods still can't pull their image"; exit 1
      fi
      echo "PASS — frontend is pulling a real image and serving. Tags are immutable promises; typos aren't."
---
