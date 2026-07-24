---
kind: lesson
title: 'Gatekeeper: the policy that rejected nothing'
description: |
  OPA Gatekeeper is installed and a Constraint says "no :latest tags" — yet
  nginx:latest sails right in. The Rego in the ConstraintTemplate reads a
  field that doesn't exist on a Pod, so the violation never fires. Fix the
  policy so it actually denies, without denying everything else.
name: opa-gatekeeper
slug: opa-gatekeeper
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
      # Gatekeeper, pinned v3.23.0 official manifest (idempotent re-apply).
      kubectl apply -f https://raw.githubusercontent.com/open-policy-agent/gatekeeper/v3.23.0/deploy/gatekeeper.yaml
      kubectl -n gatekeeper-system rollout status deploy/gatekeeper-controller-manager --timeout=180s
      kubectl -n gatekeeper-system rollout status deploy/gatekeeper-audit --timeout=120s
      # ConstraintTemplate with the shipped Rego bug: pods have no .image
      # field at the object root — the violation can never fire.
      kubectl apply -f - <<'YAML'
      apiVersion: templates.gatekeeper.sh/v1
      kind: ConstraintTemplate
      metadata:
        name: k8sdenylatest
      spec:
        crd:
          spec:
            names:
              kind: K8sDenyLatest
        targets:
          - target: admission.k8s.gatekeeper.sh
            rego: |
              package k8sdenylatest

              violation[{"msg": msg}] {
                image := input.review.object.image
                endswith(image, ":latest")
                msg := sprintf("image %v uses the :latest tag — pin a version", [image])
              }
      YAML
      # Gatekeeper generates the constraint CRD asynchronously — wait for it
      # to exist before waiting for it to be Established.
      for i in $(seq 1 45); do
        kubectl get crd k8sdenylatest.constraints.gatekeeper.sh >/dev/null 2>&1 && break
        sleep 2
      done
      kubectl wait --for=condition=Established "crd/k8sdenylatest.constraints.gatekeeper.sh" --timeout=90s
      kubectl apply -f - <<'YAML'
      apiVersion: constraints.gatekeeper.sh/v1beta1
      kind: K8sDenyLatest
      metadata:
        name: deny-latest-tag
      spec:
        enforcementAction: deny
        match:
          kinds:
            - apiGroups: [""]
              kinds: ["Pod"]
          namespaces: ["kubelings"]
      YAML
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl get constrainttemplate k8sdenylatest >/dev/null 2>&1; then
        echo "not yet: ConstraintTemplate k8sdenylatest is gone — fix the Rego, don't delete the policy"; exit 1
      fi
      if ! kubectl get k8sdenylatest deny-latest-tag >/dev/null 2>&1; then
        echo "not yet: Constraint deny-latest-tag is gone — the deny must stay in force"; exit 1
      fi
      # server-side dry-run exercises admission webhooks without persisting.
      if kubectl -n "$NS" run gk-probe-latest --image=nginx:latest --restart=Never --dry-run=server >/dev/null 2>&1; then
        echo "not yet: a nginx:latest pod is still admitted — the violation never fires; what object path does the Rego read?"; exit 1
      fi
      if ! kubectl -n "$NS" run gk-probe-pinned --image=nginx:1.25-alpine --restart=Never --dry-run=server >/dev/null 2>&1; then
        echo "not yet: a pinned-tag pod is being rejected — the policy must deny only :latest"; exit 1
      fi
      echo "PASS — :latest denied with a useful message, pinned tags admitted. The Rego finally reads the field that exists."
---
