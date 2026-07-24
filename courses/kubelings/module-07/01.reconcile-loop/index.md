---
kind: lesson
title: 'The reconcile loop: why deleted pods come back'
description: |
  Delete a pod and it reappears in seconds — not magic, but a controller
  comparing desired state to actual and acting on the difference. Watch the
  ReplicaSet controller heal a Deployment in real time, and learn the single idea
  the whole control plane is built on.
name: reconcile-loop
slug: reconcile-loop
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
        name: resilient
      spec:
        replicas: 3
        selector:
          matchLabels: {app: resilient}
        template:
          metadata:
            labels: {app: resilient}
          spec:
            containers:
              - name: web
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/resilient --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # The point of the lesson: after the learner deletes a pod, the controller
      # must have restored 3/3 — and the ReplicaSet must have observed >3 pods
      # created over its lifetime (i.e. at least one replacement happened).
      desired=$(kubectl -n "$NS" get deploy resilient -o jsonpath='{.spec.replicas}' 2>/dev/null)
      ready=$(kubectl -n "$NS" get deploy resilient -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
      if [ "${ready:-0}" -lt "${desired:-3}" ]; then
        echo "not yet: resilient is ${ready:-0}/${desired:-3} — let the controller finish healing"; exit 1
      fi
      rs=$(kubectl -n "$NS" get rs -l app=resilient -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
      # Fully-labeled pods created counter climbs each time a pod is (re)created.
      created=$(kubectl -n "$NS" get rs "$rs" -o jsonpath='{.status.replicas}' 2>/dev/null)
      events=$(kubectl -n "$NS" get events --field-selector reason=SuccessfulCreate 2>/dev/null | grep -c "$rs" || true)
      if [ "${events:-0}" -lt 4 ]; then
        echo "not yet: delete a 'resilient' pod and watch it come back — the controller must recreate at least one (SuccessfulCreate events so far: ${events:-0}, need ≥4 = 3 initial + ≥1 replacement)"; exit 1
      fi
      echo "PASS — you watched reconciliation heal the diff. Delete → observe → recreate is ALL Kubernetes does, everywhere."
---
