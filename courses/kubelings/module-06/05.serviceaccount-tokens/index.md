---
kind: lesson
title: 'The token in every pod (and who''s using yours)'
description: |
  Every pod in the namespace is carrying an API credential nobody asked for —
  the default ServiceAccount token, automounted since day one. Give the one
  workload that needs API access its own identity, and take the credential away
  from everything that doesn't.
name: serviceaccount-tokens
slug: serviceaccount-tokens
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
      kubectl -n "$NS" delete deploy audit-agent web --ignore-not-found
      kubectl -n "$NS" delete sa audit-agent --ignore-not-found
      kubectl -n "$NS" delete role pod-reader --ignore-not-found
      kubectl -n "$NS" delete rolebinding audit-agent-reads-pods --ignore-not-found
      # Reset the default SA in case a previous run turned automount off.
      kubectl -n "$NS" patch serviceaccount default --type=merge \
        -p '{"automountServiceAccountToken": null}' || true
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: web
      spec:
        replicas: 2
        selector:
          matchLabels: {app: web}
        template:
          metadata:
            labels: {app: web}
          spec:
            containers:
              - name: web
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      ---
      # The audit agent genuinely needs to list pods — but it's running as the
      # default ServiceAccount, and its RoleBinding points at an identity that
      # doesn't exist yet.
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: audit-agent
      spec:
        replicas: 1
        selector:
          matchLabels: {app: audit-agent}
        template:
          metadata:
            labels: {app: audit-agent}
          spec:
            containers:
              - name: agent
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      ---
      apiVersion: rbac.authorization.k8s.io/v1
      kind: Role
      metadata:
        name: pod-reader
      rules:
        - apiGroups: [""]
          resources: ["pods"]
          verbs: ["get", "list"]
      ---
      apiVersion: rbac.authorization.k8s.io/v1
      kind: RoleBinding
      metadata:
        name: audit-agent-reads-pods
      subjects:
        - kind: ServiceAccount
          name: audit-agent
          namespace: kubelings
      roleRef:
        kind: Role
        name: pod-reader
        apiGroup: rbac.authorization.k8s.io
      YAML
      kubectl -n "$NS" rollout status deploy/web --timeout=120s
      kubectl -n "$NS" rollout status deploy/audit-agent --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get sa audit-agent >/dev/null 2>&1; then
        echo "not yet: no ServiceAccount named audit-agent — the RoleBinding's subject points at an identity that doesn't exist"; exit 1
      fi
      sa=$(kubectl -n "$NS" get deploy audit-agent -o jsonpath='{.spec.template.spec.serviceAccountName}' 2>/dev/null)
      if [ "$sa" != "audit-agent" ]; then
        echo "not yet: audit-agent still runs as ServiceAccount '${sa:-default}' — set serviceAccountName in the pod template"; exit 1
      fi
      if [ "$(kubectl auth can-i list pods --as="system:serviceaccount:$NS:audit-agent" -n "$NS" 2>/dev/null)" != "yes" ]; then
        echo "not yet: system:serviceaccount:$NS:audit-agent can't list pods — check the Role and RoleBinding"; exit 1
      fi
      automount=$(kubectl -n "$NS" get sa default -o jsonpath='{.automountServiceAccountToken}' 2>/dev/null)
      if [ "$automount" != "false" ]; then
        echo "not yet: the default ServiceAccount still automounts its token — every pod in the namespace is carrying a credential it doesn't need"; exit 1
      fi
      desired=$(kubectl -n "$NS" get deploy audit-agent -o jsonpath='{.spec.replicas}')
      avail=$(kubectl -n "$NS" get deploy audit-agent -o jsonpath='{.status.availableReplicas}')
      if [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: audit-agent is ${avail:-0}/$desired — did the rollout with the new ServiceAccount finish?"; exit 1
      fi
      echo "PASS — one workload, one identity, one narrow grant; and the default SA hands out no tokens. That's the whole model."
---
