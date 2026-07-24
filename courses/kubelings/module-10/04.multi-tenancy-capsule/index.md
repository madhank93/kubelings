---
kind: lesson
title: 'Capsule: the tenant that hit its walls'
description: |
  Multi-tenancy without a cluster per team: Capsule carves one cluster into
  tenants. team-alpha was sized for a single namespace back when it was a
  single service — now every attempt to add one bounces off the tenant
  quota. Resize the tenant, prove the walls still hold between teams.
name: multi-tenancy-capsule
slug: multi-tenancy-capsule
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
      # Pinned helm binary (Capsule ships no static manifests — chart only;
      # helm state here is init-owned plumbing, not learner-facing).
      HELM_VERSION=v3.16.4
      if ! command -v helm >/dev/null 2>&1; then
        os=$(uname -s | tr '[:upper:]' '[:lower:]')
        arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64|arm64) arch=arm64;; esac
        curl -fsSL "https://get.helm.sh/helm-${HELM_VERSION}-${os}-${arch}.tar.gz" -o /tmp/helm.tgz
        tar xzf /tmp/helm.tgz -C /tmp
        install "/tmp/${os}-${arch}/helm" /usr/local/bin/helm
      fi
      helm repo add projectcapsule https://projectcapsule.github.io/charts >/dev/null
      helm repo update >/dev/null
      # Capsule, pinned 0.13.9; self-generated webhook TLS (no cert-manager).
      helm upgrade --install capsule projectcapsule/capsule --version 0.13.9 \
        -n capsule-system --create-namespace \
        --set certManager.generateCertificates=false \
        --set tls.enableController=true \
        --set tls.create=true \
        --wait --timeout 240s || true
      # First boot can race its own TLS secret — one restart heals it.
      if ! kubectl -n capsule-system rollout status deploy/capsule-controller-manager --timeout=60s; then
        kubectl -n capsule-system delete pod -l app.kubernetes.io/name=capsule --wait=true
        kubectl -n capsule-system rollout status deploy/capsule-controller-manager --timeout=120s
      fi
      # Reset any previous solve so the scenario is freshly broken.
      kubectl delete ns alpha-dev --ignore-not-found --wait=true
      kubectl apply -f - <<'YAML'
      apiVersion: capsule.clastix.io/v1beta2
      kind: Tenant
      metadata:
        name: team-alpha
      spec:
        owners:
          - name: alice
            kind: User
        namespaceOptions:
          # BUG(ish): sized in the single-service era; the team outgrew it.
          quota: 1
      ---
      apiVersion: capsule.clastix.io/v1beta2
      kind: Tenant
      metadata:
        name: team-beta
      spec:
        owners:
          - name: bob
            kind: User
      YAML
      # Tenant owners act through the capsule user group — impersonation
      # stands in for real SSO identities on this playground.
      kubectl get ns alpha-legacy >/dev/null 2>&1 || \
        kubectl create ns alpha-legacy --as=alice --as-group=projectcapsule.dev
      kubectl get ns beta-dev >/dev/null 2>&1 || \
        kubectl create ns beta-dev --as=bob --as-group=projectcapsule.dev
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      if ! kubectl get tenant team-alpha >/dev/null 2>&1; then
        echo "not yet: Tenant team-alpha is gone — resize it, don't delete it"; exit 1
      fi
      owner=$(kubectl get tenant team-alpha -o jsonpath='{.spec.owners[0].name}')
      if [ "$owner" != "alice" ]; then
        echo "not yet: team-alpha's owner changed — alice must stay the tenant owner"; exit 1
      fi
      if ! kubectl get ns alpha-dev >/dev/null 2>&1; then
        echo "not yet: namespace alpha-dev doesn't exist — create it AS alice (impersonate: --as=alice --as-group=projectcapsule.dev)"; exit 1
      fi
      tenant=$(kubectl get ns alpha-dev -o jsonpath='{.metadata.labels.capsule\.clastix\.io/tenant}' 2>/dev/null)
      if [ "$tenant" != "team-alpha" ]; then
        echo "not yet: alpha-dev is not owned by tenant team-alpha — did an admin create it directly instead of alice?"; exit 1
      fi
      if [ "$(kubectl auth can-i create pods -n alpha-dev --as=alice --as-group=projectcapsule.dev 2>/dev/null)" != "yes" ]; then
        echo "not yet: alice can't create pods in alpha-dev — tenant RBAC should grant that automatically"; exit 1
      fi
      if [ "$(kubectl auth can-i get pods -n beta-dev --as=alice --as-group=projectcapsule.dev 2>/dev/null)" != "no" ]; then
        echo "not yet: alice can read team-beta's namespace — tenant isolation is broken"; exit 1
      fi
      echo "PASS — tenant resized, alice self-served a namespace, and the walls between teams still hold."
---
