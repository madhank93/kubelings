---
kind: lesson
title: 'Drill — externalTrafficPolicy: Local blackholes half your nodes'
description: |
  Synthetic drill of a failure pattern reported across many production clusters:
  a Service keeps the client IP with externalTrafficPolicy: Local, but its pods
  live on one node. Every other node in the load balancer's rotation drops the
  traffic on the floor — no endpoints, no forwarding, no error anyone can see
  from inside the cluster. Keep the client IP and make every node answer.
name: pattern-etp-blackhole
slug: pattern-etp-blackhole
createdAt: "2026-07-27"
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
      # Pin every replica onto ONE worker — the way "it was faster on that node"
      # or a leftover nodeSelector does it in real clusters.
      PIN=$(kubectl get nodes -l '!node-role.kubernetes.io/control-plane' \
              -o jsonpath='{.items[0].metadata.name}')
      [ -n "$PIN" ] || PIN=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')
      kubectl -n "$NS" delete deploy checkout --ignore-not-found
      kubectl -n "$NS" delete svc checkout --ignore-not-found
      kubectl apply -n "$NS" -f - <<YAML
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
            # BUG: all three replicas are nailed to one node. The Service below
            # only forwards on nodes that HAVE a local endpoint.
            nodeSelector:
              kubernetes.io/hostname: $PIN
            containers:
              - name: web
                image: nginx:1.27-alpine
                ports:
                  - containerPort: 80
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits:   {memory: 128Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: checkout
        annotations:
          kubelings.dev/why-local: "payments needs the real client IP for fraud scoring"
      spec:
        type: NodePort
        # Intentional and CORRECT — the client IP is a product requirement.
        externalTrafficPolicy: Local
        selector: {app: checkout}
        ports:
          - name: http
            port: 80
            targetPort: 80
            nodePort: 30090
      YAML
      kubectl -n "$NS" rollout status deploy/checkout --timeout=180s
      echo "pods pinned to $PIN; Service checkout is NodePort 30090, externalTrafficPolicy=Local"
      kubectl -n "$NS" get pods -o wide
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      etp=$(kubectl -n "$NS" get svc checkout -o jsonpath='{.spec.externalTrafficPolicy}' 2>/dev/null)
      if [ "$etp" != "Local" ]; then
        echo "not yet: externalTrafficPolicy is '${etp:-missing}'. Cluster hides the bug by SNAT-ing the client IP away — fraud scoring needs that IP. Keep Local and fix the placement"; exit 1
      fi
      np=$(kubectl -n "$NS" get svc checkout -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null)
      if [ "$np" != "30090" ]; then
        echo "not yet: Service checkout must stay a NodePort on 30090 (got '${np:-none}')"; exit 1
      fi
      nodes=$(kubectl get nodes -l '!node-role.kubernetes.io/control-plane' \
                -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.addresses[?(@.type=="InternalIP")].address}{"\n"}{end}')
      [ -n "$nodes" ] || nodes=$(kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.addresses[?(@.type=="InternalIP")].address}{"\n"}{end}')
      # Node images differ on which fetcher they ship (kind: curl, k8s-omni: both).
      fetch() { if command -v curl >/dev/null 2>&1; then curl -fsS -m 4 "$1"; else wget -q -O- --timeout=4 "$1"; fi; }
      bad=""
      while read -r name ip; do
        [ -n "$ip" ] || continue
        ok=""
        # Endpoints move as pods reschedule — give each node a few tries.
        for _ in 1 2 3 4 5 6; do
          if fetch "http://$ip:30090/" 2>/dev/null | grep -qi nginx; then ok=yes; break; fi
          sleep 5
        done
        [ -n "$ok" ] || bad="$bad $name($ip)"
      done <<EOF
      $nodes
      EOF
      if [ -n "$bad" ]; then
        echo "not yet: these nodes still blackhole :30090 —$bad. With externalTrafficPolicy: Local a node forwards ONLY to endpoints it hosts; give every node a local endpoint"; exit 1
      fi
      # Ready pods by label, not Deployment status — a DaemonSet is an equally
      # valid answer to "one endpoint per node".
      ready=$(kubectl -n "$NS" get pods -l app=checkout \
                -o jsonpath='{range .items[*]}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}' 2>/dev/null \
                | grep -c True)
      if [ "${ready:-0}" -lt 2 ]; then
        echo "not yet: only ${ready:-0} ready checkout pods — spread them, don't shrink them"; exit 1
      fi
      echo "PASS — every node answers on :30090 and the client IP still survives the trip. Local traffic policy without the blackhole."
---
