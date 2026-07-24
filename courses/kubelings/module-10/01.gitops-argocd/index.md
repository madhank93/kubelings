---
kind: lesson
title: 'Argo CD: the app that refuses to sync'
description: |
  Argo CD is installed and an Application points at the guestbook example —
  but the repo path in its spec doesn't exist, so it can't even compare
  desired vs live. Read the Application status like an operator, fix the
  source, and watch auto-sync converge it to Synced/Healthy.
name: gitops-argocd
slug: gitops-argocd
createdAt: "2026-07-13"
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
      kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
      # Argo CD, pinned v3.4.5 official manifest (idempotent re-apply).
      kubectl apply -n argocd --server-side --force-conflicts -f \
        https://raw.githubusercontent.com/argoproj/argo-cd/v3.4.5/manifests/install.yaml
      # Lab simplification: Argo CD ships NetworkPolicies tuned for real CNIs;
      # under kind's kindnet enforcement they wedge the controller's own
      # API/redis/repo-server traffic. The policies are right for prod —
      # remove them here so the lesson is about GitOps, not the CNI.
      kubectl -n argocd delete networkpolicy --all --ignore-not-found
      kubectl -n argocd rollout status deploy/argocd-repo-server --timeout=240s
      # A controller started before kube-proxy programs the service rules can
      # wedge its informers on dead conntrack entries — one restart heals it.
      if ! kubectl -n argocd rollout status statefulset/argocd-application-controller --timeout=180s; then
        kubectl -n argocd delete pod argocd-application-controller-0 --wait=true
        kubectl -n argocd rollout status statefulset/argocd-application-controller --timeout=180s
      fi
      # The Application. BUG: spec.source.path names a directory that does
      # not exist in the repo.
      kubectl apply -f - <<'YAML'
      apiVersion: argoproj.io/v1alpha1
      kind: Application
      metadata:
        name: guestbook
        namespace: argocd
      spec:
        project: default
        source:
          repoURL: https://github.com/argoproj/argocd-example-apps
          path: apps/guestbook
          targetRevision: HEAD
        destination:
          server: https://kubernetes.default.svc
          namespace: kubelings
        syncPolicy:
          automated:
            selfHeal: true
          syncOptions:
            - CreateNamespace=true
      YAML
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    timeout_seconds: 300
    run: |
      NS=kubelings
      if ! kubectl -n argocd get application guestbook >/dev/null 2>&1; then
        echo "not yet: Application guestbook is gone — fix its source, don't delete it"; exit 1
      fi
      sync=$(kubectl -n argocd get application guestbook -o jsonpath='{.status.sync.status}' 2>/dev/null)
      if [ "$sync" != "Synced" ]; then
        echo "not yet: sync status is '${sync:-Unknown}' — read .status.conditions: can Argo CD even find the source path?"; exit 1
      fi
      health=$(kubectl -n argocd get application guestbook -o jsonpath='{.status.health.status}' 2>/dev/null)
      if [ "$health" != "Healthy" ]; then
        echo "not yet: health is '${health:-Unknown}' — synced but not healthy means the workload itself isn't coming up"; exit 1
      fi
      if ! kubectl -n "$NS" get deploy guestbook-ui >/dev/null 2>&1; then
        echo "not yet: guestbook-ui deployment missing in $NS — where did the manifests land?"; exit 1
      fi
      echo "PASS — Application Synced and Healthy; git said guestbook, the cluster now agrees."
---
