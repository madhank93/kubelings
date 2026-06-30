---
kind: challenge

title: "Fix the Rolling Update: unsafe maxSurge/maxUnavailable"
description: |
  A web Deployment ships every release with a full outage — its rollout strategy
  tears down all pods at once and never surges new ones. Fix the rolling update
  strategy so deploys stay available, then confirm the Deployment is healthy.

categories:
- kubernetes

tagz:
- cka
- ckad
- workloads
- rollout

difficulty: easy

createdAt: 2026-06-30

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
        name: web
        labels: {app: web}
      spec:
        replicas: 3
        strategy:
          type: RollingUpdate
          rollingUpdate:
            maxSurge: 0            # BUG: cannot add a new pod before removing old
            maxUnavailable: 100%   # BUG: allows every pod down at once -> outage
        selector:
          matchLabels: {app: web}
        template:
          metadata:
            labels: {app: web}
          spec:
            containers:
              - name: web
                image: ghcr.io/iximiuz/labs/nginx:alpine
                ports: [{containerPort: 80}]
                readinessProbe:
                  httpGet: {path: /, port: 80}
                  initialDelaySeconds: 1
                  periodSeconds: 2
      YAML
      kubectl -n "$NS" rollout status deploy/web --timeout=120s

  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      surge=$(kubectl -n "$NS" get deploy web -o jsonpath='{.spec.strategy.rollingUpdate.maxSurge}')
      unavail=$(kubectl -n "$NS" get deploy web -o jsonpath='{.spec.strategy.rollingUpdate.maxUnavailable}')
      if [ "$surge" = "0" ] || [ "$surge" = "0%" ]; then
        echo "not yet: maxSurge is $surge — rollout can never add a new pod"; exit 1
      fi
      if [ "$unavail" = "100%" ]; then
        echo "not yet: maxUnavailable is 100% — a deploy can take every pod down at once"; exit 1
      fi
      desired=$(kubectl -n "$NS" get deploy web -o jsonpath='{.spec.replicas}')
      avail=$(kubectl -n "$NS" get deploy web -o jsonpath='{.status.availableReplicas}')
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: web Deployment not fully available (${avail:-0}/$desired)"; exit 1
      fi
      echo "PASS — rolling update is safe (maxSurge=$surge, maxUnavailable=$unavail) and web is healthy."
---

## The situation

The `web` Deployment in the `kubelings` namespace serves production traffic, but
every release causes a brief total outage. Its `RollingUpdate` strategy is set to
`maxSurge: 0` and `maxUnavailable: 100%` — so on each rollout Kubernetes is allowed
to terminate **all** pods before any replacement is Ready, and is **not** allowed
to surge a single extra pod to cover the gap.

## Your task

Make `web`'s rolling update **zero-downtime**:

1. Set `maxSurge` so at least one new pod can start before old ones go away.
2. Set `maxUnavailable` so the whole fleet can't be taken down at once.
3. Keep the Deployment Available (all 3 replicas Ready).

```sh
kubectl -n kubelings get deploy web -o yaml | less
```

<details>
<summary>Hint</summary>

Edit the strategy, e.g.:

```sh
kubectl -n kubelings patch deploy web --type=merge -p \
  '{"spec":{"strategy":{"rollingUpdate":{"maxSurge":1,"maxUnavailable":"25%"}}}}'
```

`maxSurge` ≥ 1 lets a replacement come up first; `maxUnavailable` < 100% keeps
capacity during the roll.

</details>
