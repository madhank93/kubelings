---
kind: lesson
title: 'Kyverno: the policy that blocked kube-system'
description: |
  A Kyverno ClusterPolicy in Enforce mode requires an `app` label on every
  pod — in every namespace, every team, cluster-wide. Only Kyverno's default
  resourceFilters kept it from bricking kube-system too. Scope the policy to
  where it belongs, add a mutation that fixes pods instead of rejecting
  them, and keep one hard deny.
name: kyverno-policies
slug: kyverno-policies
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
      # Kyverno, pinned v1.18.2 official manifest (idempotent re-apply).
      kubectl apply -f https://github.com/kyverno/kyverno/releases/download/v1.18.2/install.yaml --server-side --force-conflicts
      kubectl -n kyverno rollout status deploy/kyverno-admission-controller --timeout=180s
      # The footgun: Enforce + match every Pod in every namespace.
      kubectl apply -f - <<'YAML'
      apiVersion: kyverno.io/v1
      kind: ClusterPolicy
      metadata:
        name: pod-guardrails
      spec:
        validationFailureAction: Enforce
        background: false
        rules:
          - name: require-app-label
            match:
              any:
                - resources:
                    kinds: ["Pod"]
            validate:
              message: "every pod must carry an app label"
              pattern:
                metadata:
                  labels:
                    app: "?*"
      YAML
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl get clusterpolicy pod-guardrails >/dev/null 2>&1; then
        echo "not yet: ClusterPolicy pod-guardrails is gone — fix the policy, don't delete it"; exit 1
      fi
      # 1. other teams' namespaces must be out of the blast radius
      if ! kubectl -n default run kyv-probe-scope --image=registry.k8s.io/pause:3.9 --restart=Never --dry-run=server >/dev/null 2>&1; then
        echo "not yet: a plain pod in the default namespace is still rejected — scope the policy to kubelings instead of matching every namespace"; exit 1
      fi
      # 2. label-less pods in kubelings get fixed, not rejected
      lbl=$(kubectl -n "$NS" run kyv-probe-mutate --image=nginx:1.25-alpine --restart=Never --dry-run=server -o jsonpath='{.metadata.labels.app}' 2>/dev/null)
      if [ -z "$lbl" ]; then
        echo "not yet: a label-less pod in kubelings comes back without an app label — add a mutate rule that injects a default"; exit 1
      fi
      # 3. one hard deny stays hard
      if kubectl -n "$NS" run kyv-probe-latest --image=nginx:latest --restart=Never --dry-run=server >/dev/null 2>&1; then
        echo "not yet: nginx:latest is admitted in kubelings — keep a validate rule that denies :latest tags"; exit 1
      fi
      echo "PASS — policy scoped to its own namespace, mutation fixes what validation used to reject, and :latest stays denied."
---
