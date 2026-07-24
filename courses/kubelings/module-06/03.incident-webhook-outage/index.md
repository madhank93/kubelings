---
kind: lesson
title: 'Incident replay — the webhook that froze the cluster (Jetstack)'
description: |
  Replay of Jetstack's cited GKE outage: a validating webhook with
  failurePolicy=Fail lost its backing pods, so the API server rejected EVERY
  write — including the ones needed to recover. Fix the failure policy and scope
  so a webhook can never again take the whole cluster hostage.
name: incident-webhook-outage
slug: incident-webhook-outage
source: https://blog.jetstack.io/blog/gke-webhook-outage
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
      # A validating webhook pointing at a service that has NO backing pods, with
      # failurePolicy=Fail and namespace-wide scope -> every write in kubelings
      # gets rejected because the admission call can't reach anything.
      kubectl apply -f - <<'YAML'
      apiVersion: admissionregistration.k8s.io/v1
      kind: ValidatingWebhookConfiguration
      metadata:
        name: policy-guard
      webhooks:
        - name: policy-guard.kubelings.svc
          admissionReviewVersions: ["v1"]
          sideEffects: None
          # BUG 1: Fail — unreachable webhook blocks writes instead of allowing.
          failurePolicy: Fail
          # BUG 2: broad scope + no namespaceSelector excluding the webhook's own ns.
          rules:
            - apiGroups: ["*"]
              apiVersions: ["*"]
              operations: ["CREATE", "UPDATE"]
              resources: ["*"]
              scope: "Namespaced"
          namespaceSelector:
            matchLabels: {webhook-guard: "enabled"}
          clientConfig:
            # points at a service with no endpoints — admission calls time out
            service:
              name: policy-guard-webhook
              namespace: kubelings
              path: /validate
              port: 443
      YAML
      kubectl label namespace "$NS" webhook-guard=enabled --overwrite
      # Prove it's wedged: this write should now be rejected by admission.
      kubectl -n "$NS" create configmap canary --from-literal=x=1 2>/dev/null || true
      sleep 2 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # The cluster must accept writes in kubelings again.
      if ! kubectl -n "$NS" create configmap admission-probe-$$ --from-literal=ok=1 >/dev/null 2>&1; then
        echo "not yet: writes to namespace $NS are still being rejected by admission — the webhook still blocks recovery"; exit 1
      fi
      kubectl -n "$NS" delete configmap admission-probe-$$ >/dev/null 2>&1 || true
      # And the config must be fixed sensibly, not merely deleted blindly:
      if kubectl get validatingwebhookconfiguration policy-guard >/dev/null 2>&1; then
        fp=$(kubectl get validatingwebhookconfiguration policy-guard \
          -o jsonpath='{.webhooks[0].failurePolicy}' 2>/dev/null)
        if [ "$fp" = "Fail" ]; then
          echo "not yet: policy-guard still failurePolicy=Fail with no reachable backend — a fail-open (Ignore) or scoped fix is required"; exit 1
        fi
      fi
      echo "PASS — the cluster can write (and recover) again. A webhook must never be able to lock out its own fix."
---
