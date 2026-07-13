---
kind: lesson
title: 'Drill — the readiness probe that flaps'
description: |
  Synthetic drill of a failure pattern reported across many production
  clusters: a service throws intermittent 502s because its pods cycle in and
  out of the endpoint pool every few seconds. The readiness probe is wired to
  a hair trigger — one blip, one second, and the pod is yanked. Tune the
  probe so it tolerates blips without lying about health.
name: pattern-readiness-flap
slug: pattern-readiness-flap
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
      kind: Deployment
      metadata:
        name: search-api
        labels: {app: search-api}
      spec:
        replicas: 2
        selector:
          matchLabels: {app: search-api}
        template:
          metadata:
            labels: {app: search-api}
          spec:
            containers:
              - name: search-api
                image: busybox:1.36
                command: ["sh", "-c", "while true; do sleep 30; done"]
                readinessProbe:
                  # The health check occasionally runs slow under load —
                  # modeled here as: fails whenever the clock second is
                  # divisible by 3 (a blip roughly 1-in-3 checks, never
                  # twice in a row).
                  exec:
                    command: ["sh", "-c", "test $(( $(date +%s) % 3 )) -ne 0"]
                  # BUG: hair trigger — one blip = out of the endpoint pool.
                  periodSeconds: 1
                  failureThreshold: 1
                  timeoutSeconds: 1
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      ---
      apiVersion: v1
      kind: Service
      metadata:
        name: search-api
      spec:
        selector: {app: search-api}
        ports: [{port: 80, targetPort: 8080}]
      YAML
      # Pods flip Ready/NotReady by design — wait only for them to exist & run.
      kubectl -n "$NS" rollout status deploy/search-api --timeout=180s || true
      kubectl -n "$NS" wait --for=jsonpath='{.status.phase}'=Running pod -l app=search-api --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      ft=$(kubectl -n "$NS" get deploy search-api -o jsonpath='{.spec.template.spec.containers[0].readinessProbe.failureThreshold}' 2>/dev/null)
      ps=$(kubectl -n "$NS" get deploy search-api -o jsonpath='{.spec.template.spec.containers[0].readinessProbe.periodSeconds}' 2>/dev/null)
      if [ -z "$ft" ] || [ -z "$ps" ]; then
        echo "not yet: search-api has no readiness probe — removing the probe is not tuning it"; exit 1
      fi
      if [ "$ft" -lt 3 ]; then
        echo "not yet: failureThreshold is $ft — one or two blips still evict the pod; give it headroom (>= 3)"; exit 1
      fi
      if [ "$ps" -lt 5 ]; then
        echo "not yet: periodSeconds is $ps — probing every second amplifies every blip; slow down (>= 5)"; exit 1
      fi
      if ! kubectl -n "$NS" rollout status deploy/search-api --timeout=120s >/dev/null 2>&1; then
        echo "not yet: search-api rollout not complete — new pods aren't settling Ready"; exit 1
      fi
      ready=$(kubectl -n "$NS" get endpoints search-api -o jsonpath='{.subsets[0].addresses[*].ip}' 2>/dev/null | wc -w)
      if [ "$ready" -lt 2 ]; then
        echo "not yet: only $ready/2 pods in the search-api endpoint pool — still churning"; exit 1
      fi
      echo "PASS — probe tolerates blips, both pods hold steady in the endpoint pool. No more 502 roulette."
---
