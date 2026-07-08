---
kind: lesson
title: 'Service types: open a door to the outside'
description: |
  The demo works inside the cluster, but the client wants to hit it from their
  laptop. Promote the ClusterIP Service to NodePort on 30080 and understand the
  ladder: ClusterIP → NodePort → LoadBalancer — what each adds, and what each
  costs.
name: nodeport-vs-clusterip
slug: nodeport-vs-clusterip
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
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: demo
      spec:
        replicas: 2
        selector:
          matchLabels: {app: demo}
        template:
          metadata:
            labels: {app: demo}
          spec:
            containers:
              - name: demo
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: demo
      spec:
        type: ClusterIP
        selector: {app: demo}
        ports: [{port: 80, targetPort: 80}]
      YAML
      kubectl -n "$NS" rollout status deploy/demo --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      t=$(kubectl -n "$NS" get svc demo -o jsonpath='{.spec.type}' 2>/dev/null)
      if [ "$t" != "NodePort" ]; then
        echo "not yet: Service 'demo' is type ${t:-missing}, needs NodePort"; exit 1
      fi
      np=$(kubectl -n "$NS" get svc demo -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null)
      if [ "$np" != "30080" ]; then
        echo "not yet: nodePort is ${np:-unset}, expected 30080"; exit 1
      fi
      # Hit a node's IP:30080 — proves the door is open on the node network.
      nip=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null)
      if ! wget -q -O- --timeout=5 "http://$nip:30080/" 2>/dev/null | grep -qi nginx; then
        echo "not yet: http://$nip:30080/ does not answer from the node network"; exit 1
      fi
      echo "PASS — every node now forwards :30080 to demo pods. That's NodePort: simple, port-limited, node-coupled."
---
