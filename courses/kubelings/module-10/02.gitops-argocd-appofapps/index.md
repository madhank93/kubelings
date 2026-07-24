---
kind: lesson
title: 'App of apps: one bad child in the fleet'
description: |
  Three Applications deploy the shop's frontend stack in sync-wave order —
  and the fleet is stuck: one child pins its targetRevision to a branch
  that doesn't exist in the repo. Triage a multi-app rollout, fix the bad
  revision, and get every Application to Synced/Healthy.
name: gitops-argocd-appofapps
slug: gitops-argocd-appofapps
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
      kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
      kubectl apply -n argocd --server-side --force-conflicts -f \
        https://raw.githubusercontent.com/argoproj/argo-cd/v3.4.5/manifests/install.yaml
      # Lab simplification: kindnet enforces Argo CD's shipped NetworkPolicies
      # badly and wedges the controller — right for prod, wrong for this lab.
      kubectl -n argocd delete networkpolicy --all --ignore-not-found
      kubectl -n argocd rollout status deploy/argocd-repo-server --timeout=240s
      if ! kubectl -n argocd rollout status statefulset/argocd-application-controller --timeout=180s; then
        kubectl -n argocd delete pod argocd-application-controller-0 --wait=true
        kubectl -n argocd rollout status statefulset/argocd-application-controller --timeout=180s
      fi
      # The fleet: three child apps in sync-wave order. In a real setup a
      # parent Application renders these from a git directory (the actual
      # app-of-apps); the children and their failure modes are identical.
      kubectl apply -f - <<'YAML'
      apiVersion: argoproj.io/v1alpha1
      kind: Application
      metadata:
        name: shop-backend
        namespace: argocd
        annotations:
          argocd.argoproj.io/sync-wave: "0"
      spec:
        project: default
        source:
          repoURL: https://github.com/argoproj/argocd-example-apps
          path: guestbook
          targetRevision: HEAD
        destination:
          server: https://kubernetes.default.svc
          namespace: shop-backend
        syncPolicy:
          automated: {selfHeal: true}
          syncOptions: [CreateNamespace=true]
      ---
      apiVersion: argoproj.io/v1alpha1
      kind: Application
      metadata:
        name: shop-frontend
        namespace: argocd
        annotations:
          argocd.argoproj.io/sync-wave: "1"
      spec:
        project: default
        source:
          repoURL: https://github.com/argoproj/argocd-example-apps
          path: kustomize-guestbook
          # BUG: this branch does not exist — someone typed the release
          # convention from their previous company.
          targetRevision: stable
        destination:
          server: https://kubernetes.default.svc
          namespace: shop-frontend
        syncPolicy:
          automated: {selfHeal: true}
          syncOptions: [CreateNamespace=true]
      ---
      apiVersion: argoproj.io/v1alpha1
      kind: Application
      metadata:
        name: shop-reports
        namespace: argocd
        annotations:
          argocd.argoproj.io/sync-wave: "2"
      spec:
        project: default
        source:
          repoURL: https://github.com/argoproj/argocd-example-apps
          path: helm-guestbook
          targetRevision: HEAD
        destination:
          server: https://kubernetes.default.svc
          namespace: shop-reports
        syncPolicy:
          automated: {selfHeal: true}
          syncOptions: [CreateNamespace=true]
      YAML
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    timeout_seconds: 300
    run: |
      for app in shop-backend shop-frontend shop-reports; do
        if ! kubectl -n argocd get application "$app" >/dev/null 2>&1; then
          echo "not yet: Application $app is gone — fix the fleet, don't shrink it"; exit 1
        fi
        sync=$(kubectl -n argocd get application "$app" -o jsonpath='{.status.sync.status}' 2>/dev/null)
        health=$(kubectl -n argocd get application "$app" -o jsonpath='{.status.health.status}' 2>/dev/null)
        if [ "$sync" != "Synced" ] || [ "$health" != "Healthy" ]; then
          echo "not yet: $app is ${sync:-Unknown}/${health:-Unknown} — 'kubectl -n argocd get applications' shows the whole fleet at a glance"; exit 1
        fi
      done
      rev=$(kubectl -n argocd get application shop-frontend -o jsonpath='{.spec.source.targetRevision}')
      if [ "$rev" = "stable" ]; then
        echo "not yet: shop-frontend still pins targetRevision: stable"; exit 1
      fi
      echo "PASS — all three Applications Synced/Healthy; the fleet reads green in one kubectl."
---
