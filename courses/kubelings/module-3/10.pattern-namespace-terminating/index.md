---
kind: lesson
title: 'Drill — the namespace stuck Terminating'
description: |
  Synthetic drill of a failure pattern reported across many production
  clusters: a namespace has been Terminating for ten minutes. A custom
  resource inside it carries a finalizer whose controller was uninstalled
  long ago — nothing will ever clear it. Find the blocker, and use the
  finalizer-strip correctly for once: on an object whose owner is
  permanently gone.
name: pattern-namespace-terminating
slug: pattern-namespace-terminating
createdAt: "2026-07-13"
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
      NS_VICTIM=team-legacy
      # Clean up any previous run: release finalizers, wait the ns out.
      if kubectl get ns "$NS_VICTIM" >/dev/null 2>&1; then
        for w in $(kubectl get widgets.kubelings.dev -n "$NS_VICTIM" -o name 2>/dev/null); do
          kubectl patch "$w" -n "$NS_VICTIM" --type=merge -p '{"metadata":{"finalizers":null}}' >/dev/null 2>&1 || true
        done
        kubectl delete ns "$NS_VICTIM" --ignore-not-found --wait=true --timeout=90s >/dev/null 2>&1 || true
      fi
      # A CRD whose controller was "uninstalled" — the CRD and its objects
      # remain, nothing processes their finalizers anymore.
      kubectl apply -f - <<'YAML'
      apiVersion: apiextensions.k8s.io/v1
      kind: CustomResourceDefinition
      metadata:
        name: widgets.kubelings.dev
      spec:
        group: kubelings.dev
        scope: Namespaced
        names:
          plural: widgets
          singular: widget
          kind: Widget
        versions:
          - name: v1
            served: true
            storage: true
            schema:
              openAPIV3Schema:
                type: object
                x-kubernetes-preserve-unknown-fields: true
      YAML
      kubectl create namespace "$NS_VICTIM" --dry-run=client -o yaml | kubectl apply -f -
      kubectl apply -f - <<'YAML'
      apiVersion: kubelings.dev/v1
      kind: Widget
      metadata:
        name: legacy-exporter
        namespace: team-legacy
        finalizers:
          - kubelings.dev/widget-cleanup
      YAML
      # The teardown: delete the namespace. It will stick on the Widget.
      kubectl delete ns "$NS_VICTIM" --wait=false
      sleep 3 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      if kubectl get ns team-legacy >/dev/null 2>&1; then
        if kubectl get widgets.kubelings.dev -n team-legacy --no-headers 2>/dev/null | grep -q .; then
          echo "not yet: team-legacy still Terminating — a resource inside it refuses to die. Which one, and what's holding it?"; exit 1
        fi
        echo "not yet: team-legacy still exists — give the namespace controller a few seconds after the blocker clears, then re-verify"; exit 1
      fi
      echo "PASS — blocker found, orphaned finalizer released, namespace gone. Strip finalizers only when the controller is provably dead — like this one."
---
