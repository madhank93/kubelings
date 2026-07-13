---
kind: lesson
title: 'No shell, no exec, no problem'
description: |
  A distroless production pod is misbehaving and `kubectl exec` greets you with
  "no such file or directory" — there is no shell in the image. Attach an
  ephemeral debug container with `kubectl debug`, read the app's filesystem
  through the shared PID namespace, and fix the bad config it's choking on.
name: ephemeral-containers
slug: ephemeral-containers
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
      # Fresh pod each init: ephemeralContainers can't be removed once added.
      kubectl -n "$NS" delete pod orders-api --ignore-not-found --wait=true
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: orders-code
      data:
        main.py: |
          import json, sys, time
          while True:
              try:
                  with open("/etc/app/config.json") as f:
                      cfg = json.load(f)
                  if cfg.get("mode") == "production":
                      print("config ok — serving orders", flush=True)
                  else:
                      print("fatal: config invalid, refusing to serve (retrying)", flush=True)
              except Exception as e:
                  print(f"fatal: cannot read config: {e}", flush=True)
              time.sleep(10)
      ---
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: orders-config
      data:
        # BUG: mode must be "production"; nobody can see this from the logs.
        config.json: |
          {"mode": "debug", "flush_interval": 5}
      ---
      apiVersion: v1
      kind: Pod
      metadata:
        name: orders-api
        labels: {app: orders-api}
      spec:
        containers:
          - name: orders-api
            image: gcr.io/distroless/python3-debian12:latest
            args: ["/app/main.py"]
            volumeMounts:
              - {name: code, mountPath: /app}
              - {name: config, mountPath: /etc/app}
            resources:
              requests: {cpu: 10m, memory: 32Mi}
              limits: {memory: 128Mi}
        volumes:
          - name: code
            configMap: {name: orders-code}
          - name: config
            configMap: {name: orders-config}
      YAML
      kubectl -n "$NS" wait --for=condition=Ready pod/orders-api --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      eph=$(kubectl -n "$NS" get pod orders-api -o jsonpath='{.spec.ephemeralContainers}' 2>/dev/null)
      if [ -z "$eph" ]; then
        echo "not yet: no ephemeral container on orders-api — this lesson is about 'kubectl debug', use it"; exit 1
      fi
      phase=$(kubectl -n "$NS" get pod orders-api -o jsonpath='{.status.phase}' 2>/dev/null)
      if [ "$phase" != "Running" ]; then
        echo "not yet: orders-api is ${phase:-gone} — it should stay Running"; exit 1
      fi
      if ! kubectl -n "$NS" logs orders-api -c orders-api --tail=3 | grep -q "config ok"; then
        echo "not yet: app still rejects its config — fix the ConfigMap; kubelet takes up to ~1 min to project the change, then re-verify"; exit 1
      fi
      echo "PASS — debugged a shell-less container via an ephemeral container and fixed the config it was choking on."
---
