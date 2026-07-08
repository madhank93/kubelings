---
kind: lesson
title: 'Incident replay — the exposed dashboard (JW Player''s cryptominer)'
description: |
  Replay of JW Player's cited incident: an internal ops tool got a public
  LoadBalancer with no auth, and attackers used it to launch a cryptominer
  across the cluster. Find the pod nobody deployed, kill it, and shut the door
  it walked through.
name: incident-cryptominer
slug: incident-cryptominer
source: https://medium.com/jw-player-engineering/how-a-cryptocurrency-miner-made-its-way-onto-our-internal-kubernetes-clusters-9b09c4704205
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
      # An internal ops tool, exposed to the world with no auth (the root cause).
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: ops-console
        labels: {app: ops-console, exposure: public}
      spec:
        replicas: 1
        selector:
          matchLabels: {app: ops-console}
        template:
          metadata:
            labels: {app: ops-console}
          spec:
            containers:
              - name: console
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: ops-console
      spec:
        # BUG: internal tool, exposed to the internet, no auth in front.
        type: NodePort
        selector: {app: ops-console}
        ports: [{port: 80, targetPort: 80, nodePort: 31337}]
      YAML
      # The consequence: a workload nobody legitimately deployed.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: sys-helper
        labels: {app: sys-helper}
        annotations: {note: "not in any git repo — where did this come from?"}
      spec:
        replicas: 2
        selector:
          matchLabels: {app: sys-helper}
        template:
          metadata:
            labels: {app: sys-helper, workload: xmrig-lookalike}
          spec:
            containers:
              - name: miner
                image: busybox:1.36
                command: ["sh", "-c", "echo '[miner] hashing on stolen compute...'; while true; do :; done"]
                resources:
                  requests: {cpu: 100m, memory: 16Mi}
                  limits: {memory: 64Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/ops-console --timeout=180s
      kubectl -n "$NS" rollout status deploy/sys-helper --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # 1) The malicious workload must be gone.
      if kubectl -n "$NS" get deploy sys-helper >/dev/null 2>&1; then
        echo "not yet: the sys-helper (miner) workload is still running — evict the intruder"; exit 1
      fi
      miners=$(kubectl -n "$NS" get pods -l workload=xmrig-lookalike -o name 2>/dev/null | grep -c . || true)
      if [ "${miners:-0}" -gt 0 ]; then
        echo "not yet: miner pods still present"; exit 1
      fi
      # 2) The door must be shut: ops-console no longer publicly exposed.
      t=$(kubectl -n "$NS" get svc ops-console -o jsonpath='{.spec.type}' 2>/dev/null)
      if [ "$t" = "NodePort" ] || [ "$t" = "LoadBalancer" ]; then
        echo "not yet: ops-console is still $t (publicly reachable) — make it ClusterIP or remove the Service"; exit 1
      fi
      echo "PASS — intruder evicted AND the exposure closed. Kill the process, then kill the door — order matters."
---
