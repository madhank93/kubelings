---
kind: lesson
title: 'Node maintenance: drain like you mean it'
description: |
  A worker node needs a kernel patch. It runs a DaemonSet pod and a batch
  pod with emptyDir scratch data — the two things `kubectl drain` refuses
  to touch without explicit flags. Run the full maintenance cycle: cordon,
  drain past both refusals, "patch", uncordon, and leave a paper trail.
name: node-maintenance
slug: node-maintenance
createdAt: "2026-07-13"
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
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: DaemonSet
      metadata:
        name: node-agent
        labels: {app: node-agent}
      spec:
        selector:
          matchLabels: {app: node-agent}
        template:
          metadata:
            labels: {app: node-agent}
          spec:
            tolerations:
              - key: node-role.kubernetes.io/control-plane
                operator: Exists
                effect: NoSchedule
            containers:
              - name: agent
                image: busybox:1.36
                command: ["sh", "-c", "while true; do sleep 30; done"]
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: orders-batch
        labels: {app: orders-batch}
      spec:
        replicas: 1
        selector:
          matchLabels: {app: orders-batch}
        template:
          metadata:
            labels: {app: orders-batch}
          spec:
            containers:
              - name: batch
                image: busybox:1.36
                command: ["sh", "-c", "echo 'scratch data' > /scratch/wip.dat; while true; do sleep 30; done"]
                volumeMounts:
                  - {name: scratch, mountPath: /scratch}
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
            volumes:
              - name: scratch
                emptyDir: {}
      YAML
      kubectl -n "$NS" rollout status deploy/orders-batch --timeout=120s
      kubectl -n "$NS" rollout status ds/node-agent --timeout=120s
      # Record which node drew the short straw — that one gets "maintained".
      node=$(kubectl -n "$NS" get pods -l app=orders-batch -o jsonpath='{.items[0].spec.nodeName}')
      kubectl -n "$NS" create configmap maintenance-target --from-literal=node="$node" \
        --dry-run=client -o yaml | kubectl apply -f -
      # Clean any leftover state from a previous run.
      kubectl uncordon "$node" >/dev/null 2>&1 || true
      kubectl label node "$node" kubelings.dev/maintenance-done- >/dev/null 2>&1 || true
      echo "maintenance target: $node"
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      node=$(kubectl -n "$NS" get configmap maintenance-target -o jsonpath='{.data.node}' 2>/dev/null)
      if [ -z "$node" ]; then
        echo "not yet: configmap maintenance-target missing — re-run init"; exit 1
      fi
      podnode=$(kubectl -n "$NS" get pods -l app=orders-batch --field-selector=status.phase=Running -o jsonpath='{.items[0].spec.nodeName}' 2>/dev/null)
      if [ -z "$podnode" ]; then
        echo "not yet: orders-batch has no Running pod"; exit 1
      fi
      if [ "$podnode" = "$node" ]; then
        echo "not yet: orders-batch still runs on $node — it must be evicted to another node (cordon + drain)"; exit 1
      fi
      cordoned=$(kubectl get node "$node" -o jsonpath='{.spec.unschedulable}' 2>/dev/null)
      if [ "$cordoned" = "true" ]; then
        echo "not yet: $node is still cordoned — maintenance ends with uncordon"; exit 1
      fi
      if ! kubectl -n "$NS" get pods -l app=node-agent --field-selector="spec.nodeName=$node,status.phase=Running" --no-headers 2>/dev/null | grep -q .; then
        echo "not yet: the node-agent DaemonSet pod is not Running on $node — it should have survived (or returned after) the drain"; exit 1
      fi
      done_label=$(kubectl get node "$node" -o jsonpath='{.metadata.labels.kubelings\.dev/maintenance-done}' 2>/dev/null)
      if [ "$done_label" != "true" ]; then
        echo "not yet: label the node when finished — kubectl label node $node kubelings.dev/maintenance-done=true"; exit 1
      fi
      echo "PASS — drained past both refusals, workload rehomed, node back in service with a paper trail."
---
