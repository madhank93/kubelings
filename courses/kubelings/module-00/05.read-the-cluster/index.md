---
kind: lesson
title: 'get, describe, logs: finding out what happened'
description: |
  Three commands answer almost every "what is going on?" question: get for the
  list, describe for the story and its events, logs for what the program said.
  Use them to dig two facts out of a running app.
name: read-the-cluster
slug: read-the-cluster
createdAt: "2026-11-22"
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
      kubectl -n "$NS" delete configmap findings --ignore-not-found
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: greeter
      spec:
        replicas: 1
        selector:
          matchLabels: {app: greeter}
        template:
          metadata:
            labels: {app: greeter}
          spec:
            containers:
              - name: greeter
                image: busybox:1.36
                command: ["sh", "-c"]
                args:
                  - |
                    w=$(tr -dc a-z </dev/urandom | head -c 8)
                    echo "greeter starting"
                    echo "the secret word is $w"
                    echo "ready"
                    exec sleep 2147483647
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 32Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/greeter --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get configmap findings >/dev/null 2>&1; then
        echo "not yet: no ConfigMap 'findings' in namespace '$NS'"; exit 1
      fi
      want_word=$(kubectl -n "$NS" logs deploy/greeter 2>/dev/null | sed -n 's/^the secret word is //p' | tail -1)
      got_word=$(kubectl -n "$NS" get configmap findings -o jsonpath='{.data.word}')
      if [ -z "$want_word" ] || [ "$got_word" != "$want_word" ]; then
        echo "not yet: 'word' in findings is '$got_word' — it should be the word greeter logs"; exit 1
      fi
      want_node=$(kubectl -n "$NS" get pods -l app=greeter -o jsonpath='{.items[0].spec.nodeName}')
      got_node=$(kubectl -n "$NS" get configmap findings -o jsonpath='{.data.node}')
      if [ "$got_node" != "$want_node" ]; then
        echo "not yet: 'node' in findings is '$got_node' — it should be the node greeter's pod runs on"; exit 1
      fi
      echo "PASS — you read the logs and found the node. That loop is the rest of the course."
---
