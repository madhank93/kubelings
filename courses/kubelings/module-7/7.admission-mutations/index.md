---
kind: lesson
title: 'Admission: the API server edits your YAML before storing it'
description: |
  Apply a bare pod and read back what the cluster stored: resources you never
  wrote, a ServiceAccount you never named, volumes you never mounted. That's
  the admission chain rewriting objects in flight. Drive it yourself with a
  LimitRange — defaults in (mutation), oversized pods out (validation).
name: admission-mutations
slug: admission-mutations
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
      kubectl -n "$NS" delete limitrange defaults --ignore-not-found
      kubectl -n "$NS" delete deploy sample --ignore-not-found
      kubectl -n "$NS" delete pod bare oversized --ignore-not-found --force --grace-period=0 >/dev/null 2>&1 || true
      # The scenario: a namespace with NO LimitRange, and a "bare" pod showing
      # what admission injects even with zero policy configured.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: Pod
      metadata:
        name: bare
      spec:
        containers:
          - name: c
            image: busybox:1.36
            command: ["sh", "-c", "sleep 3600"]
      YAML
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get limitrange defaults >/dev/null 2>&1; then
        echo "not yet: no LimitRange named 'defaults' — create the namespace policy first"; exit 1
      fi
      dreq=$(kubectl -n "$NS" get limitrange defaults -o jsonpath='{.spec.limits[0].defaultRequest.cpu}' 2>/dev/null)
      dlim=$(kubectl -n "$NS" get limitrange defaults -o jsonpath='{.spec.limits[0].default.memory}' 2>/dev/null)
      if [ -z "$dreq" ] || [ -z "$dlim" ]; then
        echo "not yet: the LimitRange needs defaultRequest (cpu) and default (memory) — the mutating half"; exit 1
      fi
      maxmem=$(kubectl -n "$NS" get limitrange defaults -o jsonpath='{.spec.limits[0].max.memory}' 2>/dev/null)
      if [ -z "$maxmem" ]; then
        echo "not yet: the LimitRange has no max.memory — the validating half is missing"; exit 1
      fi
      if ! kubectl -n "$NS" get deploy sample >/dev/null 2>&1; then
        echo "not yet: create the 'sample' deployment (no resources in its spec) to see the defaults injected"; exit 1
      fi
      speccpu=$(kubectl -n "$NS" get deploy sample -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}' 2>/dev/null)
      if [ -n "$speccpu" ]; then
        echo "not yet: 'sample' declares resources in its own spec — leave it bare; the point is watching admission fill them in on the POD"; exit 1
      fi
      podcpu=$(kubectl -n "$NS" get pods -l app=sample -o jsonpath='{.items[0].spec.containers[0].resources.requests.cpu}' 2>/dev/null)
      if [ -z "$podcpu" ]; then
        echo "not yet: sample's pod has no injected CPU request — is the LimitRange in the same namespace? (defaults apply at pod-create time)"; exit 1
      fi
      avail=$(kubectl -n "$NS" get deploy sample -o jsonpath='{.status.availableReplicas}')
      if [ "${avail:-0}" -lt 1 ]; then
        echo "not yet: sample isn't running yet"; exit 1
      fi
      echo "PASS — the deployment says nothing, the pod says $podcpu: admission wrote that. Mutation in, validation at the door."
---
