---
kind: lesson
title: 'Crossplane: the composition missing its provider'
description: |
  A platform team publishes XDatabase as self-service API — claim one and a
  Composition assembles the real resources. Except every XDatabase sits
  Synced=False: the Composition emits a resource kind no installed provider
  understands. Read the composition error, install the missing provider,
  watch the claim go Ready.
name: crossplane-compositions
slug: crossplane-compositions
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
      # Pinned helm binary (Crossplane ships charts only; helm state is
      # init-owned plumbing, not learner-facing).
      HELM_VERSION=v3.16.4
      if ! command -v helm >/dev/null 2>&1; then
        os=$(uname -s | tr '[:upper:]' '[:lower:]')
        arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64|arm64) arch=arm64;; esac
        curl -fsSL "https://get.helm.sh/helm-${HELM_VERSION}-${os}-${arch}.tar.gz" -o /tmp/helm.tgz
        tar xzf /tmp/helm.tgz -C /tmp
        install "/tmp/${os}-${arch}/helm" /usr/local/bin/helm
      fi
      helm repo add crossplane-stable https://charts.crossplane.io/stable >/dev/null
      helm repo update >/dev/null
      helm upgrade --install crossplane crossplane-stable/crossplane --version 2.3.3 \
        -n crossplane-system --create-namespace --wait --timeout 240s
      # Reset any previous solve: the missing provider must stay missing.
      kubectl delete providers.pkg.crossplane.io provider-nop --ignore-not-found
      # Composition functions are v2's rendering engine — pinned.
      kubectl apply -f - <<'YAML'
      apiVersion: pkg.crossplane.io/v1
      kind: Function
      metadata:
        name: function-patch-and-transform
      spec:
        package: xpkg.crossplane.io/crossplane-contrib/function-patch-and-transform:v0.10.7
      YAML
      for i in $(seq 1 30); do
        h=$(kubectl get function function-patch-and-transform -o jsonpath='{.status.conditions[?(@.type=="Healthy")].status}' 2>/dev/null)
        [ "$h" = "True" ] && break
        sleep 4
      done
      # The platform API: XDatabase, and the Composition that implements it.
      kubectl apply -f - <<'YAML'
      apiVersion: apiextensions.crossplane.io/v2
      kind: CompositeResourceDefinition
      metadata:
        name: xdatabases.demo.kubelings.dev
      spec:
        scope: Namespaced
        group: demo.kubelings.dev
        names:
          kind: XDatabase
          plural: xdatabases
        versions:
          - name: v1alpha1
            served: true
            referenceable: true
            schema:
              openAPIV3Schema:
                type: object
                properties:
                  spec:
                    type: object
                    properties:
                      size:
                        type: string
                        default: small
      YAML
      kubectl apply -f - <<'YAML'
      apiVersion: apiextensions.crossplane.io/v1
      kind: Composition
      metadata:
        name: xdatabase-nop
      spec:
        compositeTypeRef:
          apiVersion: demo.kubelings.dev/v1alpha1
          kind: XDatabase
        mode: Pipeline
        pipeline:
          - step: create-database
            functionRef:
              name: function-patch-and-transform
            input:
              apiVersion: pt.fn.crossplane.io/v1beta1
              kind: Resources
              resources:
                - name: database
                  base:
                    apiVersion: nop.crossplane.io/v1alpha1
                    kind: NopResource
                    spec:
                      forProvider:
                        conditionAfter:
                          - conditionType: Ready
                            conditionStatus: "True"
                            time: 5s
      YAML
      # A developer claims a database. It will sit broken — that's the lesson.
      kubectl apply -f - <<'YAML'
      apiVersion: demo.kubelings.dev/v1alpha1
      kind: XDatabase
      metadata:
        name: orders-db
        namespace: kubelings
      spec:
        size: small
      YAML
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    timeout_seconds: 300
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get xdatabase orders-db >/dev/null 2>&1; then
        echo "not yet: XDatabase orders-db is gone — the claim stays; fix what renders it"; exit 1
      fi
      if ! kubectl get providers.pkg.crossplane.io provider-nop >/dev/null 2>&1; then
        echo "not yet: no provider installed — the Composition emits nop.crossplane.io resources; which provider owns that API group?"; exit 1
      fi
      synced=$(kubectl -n "$NS" get xdatabase orders-db -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null)
      if [ "$synced" != "True" ]; then
        msg=$(kubectl -n "$NS" get xdatabase orders-db -o jsonpath='{.status.conditions[?(@.type=="Synced")].message}' 2>/dev/null)
        echo "not yet: orders-db not Synced — ${msg:-composition not rendering}"; exit 1
      fi
      ready=$(kubectl -n "$NS" get xdatabase orders-db -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)
      if [ "$ready" != "True" ]; then
        echo "not yet: orders-db Synced but not Ready — composed resources still coming up; give it ~30s and re-verify"; exit 1
      fi
      echo "PASS — provider installed, composition renders, claim Ready. The platform API serves again."
---
