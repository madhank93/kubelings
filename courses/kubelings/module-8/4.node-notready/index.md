---
kind: lesson
title: 'The morning after NotReady'
description: |
  A worker went NotReady overnight — kubelet certificate expired, on-call fixed
  it at 3 a.m. and went back to bed. The node is healthy now, but half your
  capacity is still gone and pods are Pending. Learn what NotReady actually does
  to a node — taints, evictions, cordons — and how to clean up after it.
name: node-notready
slug: node-notready
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
      kubectl -n "$NS" delete deploy checkout --ignore-not-found
      # Pick a worker to be "the node that had a bad night".
      WNODE=$(kubectl get nodes --no-headers -o custom-columns=NAME:.metadata.name \
        | grep -v control-plane | head -1)
      # Clean any previous run, then label it so the workload is pinned there.
      for n in $(kubectl get nodes -o name); do
        kubectl uncordon "${n#node/}" >/dev/null 2>&1 || true
        kubectl label "${n#node/}" kubelings/rack- >/dev/null 2>&1 || true
        kubectl taint node "${n#node/}" kubelings/maintenance:NoExecute- >/dev/null 2>&1 || true
      done
      kubectl label node "$WNODE" kubelings/rack=a
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: checkout
      spec:
        replicas: 3
        selector:
          matchLabels: {app: checkout}
        template:
          metadata:
            labels: {app: checkout}
          spec:
            nodeSelector:
              kubelings/rack: "a"
            containers:
              - name: checkout
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/checkout --timeout=120s
      # The 3 a.m. incident aftermath: while the node was NotReady, on-call
      # cordoned it and added a NoExecute taint to keep pods off during the
      # kubelet cert fix — then went back to bed without removing either.
      kubectl cordon "$WNODE"
      kubectl taint node "$WNODE" kubelings/maintenance=cert-rotation:NoExecute --overwrite
      # Give the taint manager a moment to evict the pods.
      sleep 10 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      bad=$(kubectl get nodes -o jsonpath='{range .items[?(@.spec.unschedulable==true)]}{.metadata.name}{"\n"}{end}')
      if [ -n "$bad" ]; then
        echo "not yet: still cordoned: $bad — scheduling is disabled there (check kubectl get nodes)"; exit 1
      fi
      tainted=$(kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{" taints: "}{.spec.taints[*].key}{"\n"}{end}' | grep 'kubelings/maintenance' || true)
      if [ -n "$tainted" ]; then
        echo "not yet: the maintenance taint is still on ${tainted%% *} — NoExecute keeps evicting anything that lands there"; exit 1
      fi
      desired=$(kubectl -n "$NS" get deploy checkout -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy checkout -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: checkout is ${avail:-0}/${desired:-?} — pods still can't land on their rack"; exit 1
      fi
      echo "PASS — node clean, capacity back, checkout at full strength. NotReady is a lifecycle, not a status column."
---
