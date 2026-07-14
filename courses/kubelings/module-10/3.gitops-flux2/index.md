---
kind: lesson
title: 'Flux: the Kustomization that can''t find its source'
description: |
  Flux reconciles podinfo from git — except the GitRepository pins a branch
  that doesn't exist, so the artifact never materializes and the
  Kustomization downstream starves. Read Flux's condition chain source →
  kustomization, fix the ref, watch it converge.
name: gitops-flux2
slug: gitops-flux2
createdAt: "2026-07-14"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 300
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      # Flux2, pinned v2.9.2 official install manifest (controllers + CRDs;
      # no `flux bootstrap` — that wants write access to a git repo).
      kubectl apply --server-side --force-conflicts -f \
        https://github.com/fluxcd/flux2/releases/download/v2.9.2/install.yaml
      # Same lab simplification as the Argo lessons: kindnet's NetworkPolicy
      # enforcement wedges the controllers' own traffic; fine for prod CNIs.
      kubectl -n flux-system delete networkpolicy --all --ignore-not-found
      kubectl -n flux-system rollout status deploy/source-controller --timeout=180s
      kubectl -n flux-system rollout status deploy/kustomize-controller --timeout=180s
      kubectl apply -f - <<'YAML'
      apiVersion: source.toolkit.fluxcd.io/v1
      kind: GitRepository
      metadata:
        name: podinfo
        namespace: kubelings
      spec:
        interval: 1m
        url: https://github.com/stefanprodan/podinfo
        ref:
          # BUG: podinfo has no 'production' branch.
          branch: production
      ---
      apiVersion: kustomize.toolkit.fluxcd.io/v1
      kind: Kustomization
      metadata:
        name: podinfo
        namespace: kubelings
      spec:
        interval: 1m
        targetNamespace: kubelings
        sourceRef:
          kind: GitRepository
          name: podinfo
        path: ./kustomize
        prune: true
        timeout: 2m
      YAML
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    timeout_seconds: 300
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get gitrepository podinfo >/dev/null 2>&1; then
        echo "not yet: GitRepository podinfo is gone — fix the ref, don't delete the source"; exit 1
      fi
      src=$(kubectl -n "$NS" get gitrepository podinfo -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)
      if [ "$src" != "True" ]; then
        msg=$(kubectl -n "$NS" get gitrepository podinfo -o jsonpath='{.status.conditions[?(@.type=="Ready")].message}' 2>/dev/null)
        echo "not yet: GitRepository not Ready — ${msg:-no artifact}; which refs does the repo actually have?"; exit 1
      fi
      kst=$(kubectl -n "$NS" get kustomization podinfo -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)
      if [ "$kst" != "True" ]; then
        msg=$(kubectl -n "$NS" get kustomization podinfo -o jsonpath='{.status.conditions[?(@.type=="Ready")].message}' 2>/dev/null)
        echo "not yet: Kustomization not Ready — ${msg:-waiting}; flux reconciles source first, then kustomization"; exit 1
      fi
      if ! kubectl -n "$NS" get deploy podinfo >/dev/null 2>&1; then
        echo "not yet: podinfo deployment missing in $NS"; exit 1
      fi
      echo "PASS — source Ready, Kustomization Ready, podinfo running. The chain reconciles end to end."
---
