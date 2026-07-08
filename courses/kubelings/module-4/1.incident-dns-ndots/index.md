---
kind: lesson
title: 'Incident replay — the ndots:5 DNS amplifier (Zalando, Jan 2019)'
description: |
  Replay of a real production outage: Zalando's fashion store lost ALL cluster
  DNS for over an hour when retry traffic, un-cached lookups and Kubernetes'
  default ndots:5 multiplied every external hostname lookup into ~10 DNS queries,
  OOMKilling every CoreDNS pod at once. Find the amplifier in this cluster and
  defuse it.
name: incident-dns-ndots
slug: incident-dns-ndots
source: https://github.com/zalando-incubator/kubernetes-on-aws/blob/dev/docs/postmortems/jan-2019-dns-outage.md
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
      # checkout talks to an external payments API on every request.
      # Default dnsPolicy: ClusterFirst gives it ndots:5 + long search path:
      # every lookup of the external name fans out into ~10 queries.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: checkout
      spec:
        replicas: 2
        selector:
          matchLabels: {app: checkout}
        template:
          metadata:
            labels: {app: checkout}
          spec:
            containers:
              - name: checkout
                image: busybox:1.36
                command: ["sh", "-c"]
                args:
                  - |
                    echo "checkout: resolving payments.example.com in a loop (like prod retry traffic)"
                    while true; do
                      nslookup payments.example.com >/dev/null 2>&1 || true
                      sleep 2
                    done
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/checkout --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      desired=$(kubectl -n "$NS" get deploy checkout -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy checkout -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: checkout Deployment not Available (${avail:-0}/${desired:-?})"; exit 1
      fi
      pod=$(kubectl -n "$NS" get pods -l app=checkout -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
      if [ -z "$pod" ]; then
        echo "not yet: no checkout pods found"; exit 1
      fi
      resolv=$(kubectl -n "$NS" exec "$pod" -- cat /etc/resolv.conf 2>/dev/null)
      if ! grep -qE 'ndots:[12]([^0-9]|$)' <<<"$resolv"; then
        echo "not yet: checkout pods still run with the default ndots:5 — every external"
        echo "lookup fans out through the whole search path. Set dnsConfig ndots to 1 (or 2)."
        exit 1
      fi
      if ! grep -q 'search' <<<"$resolv"; then
        echo "not yet: cluster DNS search path missing — keep dnsPolicy ClusterFirst and"
        echo "tune ndots via dnsConfig.options instead of dropping cluster DNS entirely."
        exit 1
      fi
      echo "PASS — ndots tamed: external lookups now resolve in one query instead of ~10."
      echo "That amplification is exactly what OOMKilled every CoreDNS pod at Zalando."
---
