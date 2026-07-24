---
kind: lesson
title: 'Egress lockdown: the miner needs a phone line'
description: |
  The cryptominer from lesson 6.2 did its damage over outbound connections —
  and outbound is the direction nobody polices. Lock the payments namespace
  down: default-deny all egress, then allow back exactly two flows — DNS, and
  the payment gateway. Everything else, including the next miner, dials a
  dead line.
name: egress-lockdown
slug: egress-lockdown
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
      kubectl -n "$NS" delete deploy payments gateway --ignore-not-found
      kubectl -n "$NS" delete svc gateway --ignore-not-found
      kubectl -n "$NS" delete networkpolicy --all >/dev/null 2>&1 || true
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: payments
      spec:
        replicas: 2
        selector:
          matchLabels: {app: payments}
        template:
          metadata:
            labels: {app: payments}
          spec:
            containers:
              - name: payments
                image: nginx:1.27-alpine
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: gateway
      spec:
        replicas: 1
        selector:
          matchLabels: {app: gateway}
        template:
          metadata:
            labels: {app: gateway}
          spec:
            containers:
              - name: gateway
                image: nginx:1.27-alpine
                ports: [{containerPort: 80}]
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: gateway
      spec:
        selector: {app: gateway}
        ports: [{port: 80, targetPort: 80}]
      YAML
      kubectl -n "$NS" rollout status deploy/payments --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # 1. default-deny egress covering all pods in the namespace
      if ! kubectl -n "$NS" get networkpolicy egress-default-deny >/dev/null 2>&1; then
        echo "not yet: no NetworkPolicy named egress-default-deny — the floor comes first"; exit 1
      fi
      pt=$(kubectl -n "$NS" get networkpolicy egress-default-deny -o jsonpath='{.spec.policyTypes}')
      case "$pt" in *Egress*) : ;; *)
        echo "not yet: egress-default-deny doesn't list Egress in policyTypes — without that it denies nothing outbound"; exit 1 ;;
      esac
      sel=$(kubectl -n "$NS" get networkpolicy egress-default-deny -o jsonpath='{.spec.podSelector}')
      if [ "$sel" != "{}" ] && [ "$sel" != "map[]" ]; then
        echo "not yet: egress-default-deny's podSelector isn't empty — it must select every pod in the namespace"; exit 1
      fi
      rules=$(kubectl -n "$NS" get networkpolicy egress-default-deny -o jsonpath='{.spec.egress}')
      if [ -n "$rules" ] && [ "$rules" != "[]" ]; then
        echo "not yet: egress-default-deny has egress rules — deny-all means NO rules (allows live in separate policies)"; exit 1
      fi
      # 2. DNS allowed back
      if ! kubectl -n "$NS" get networkpolicy allow-dns >/dev/null 2>&1; then
        echo "not yet: no allow-dns policy — with egress denied, every name lookup in the namespace is dead"; exit 1
      fi
      dns=$(kubectl -n "$NS" get networkpolicy allow-dns -o jsonpath='{.spec.egress[*].ports[*].port}')
      case "$dns" in *53*) : ;; *)
        echo "not yet: allow-dns doesn't open port 53 — check its egress ports (UDP and TCP)"; exit 1 ;;
      esac
      # 3. payments -> gateway allowed back
      if ! kubectl -n "$NS" get networkpolicy allow-payments-to-gateway >/dev/null 2>&1; then
        echo "not yet: no allow-payments-to-gateway policy — payments still can't reach its one legitimate dependency"; exit 1
      fi
      psel=$(kubectl -n "$NS" get networkpolicy allow-payments-to-gateway -o jsonpath='{.spec.podSelector.matchLabels.app}')
      if [ "$psel" != "payments" ]; then
        echo "not yet: allow-payments-to-gateway must select the payments pods (podSelector app=payments), not everything"; exit 1
      fi
      tsel=$(kubectl -n "$NS" get networkpolicy allow-payments-to-gateway -o jsonpath='{.spec.egress[0].to[0].podSelector.matchLabels.app}')
      if [ "$tsel" != "gateway" ]; then
        echo "not yet: the egress 'to' must target the gateway pods by label"; exit 1
      fi
      echo "PASS — deny by default, allow by name. The next miner boots fine and starves: no DNS for its pool, no route out."
---
