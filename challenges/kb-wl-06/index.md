---
kind: challenge

title: "Autoscale a Deployment with an HPA (1 → 5)"
description: |
  A CPU-bound web app needs to scale out under load and back in when idle. With
  metrics-server already running, create a HorizontalPodAutoscaler that scales the
  Deployment between 1 and 5 replicas on CPU.

categories:
- kubernetes

tagz:
- cka
- workloads
- autoscaling

difficulty: medium

createdAt: 2026-06-30

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

      # Ensure metrics-server is present (HPA needs it). Idempotent.
      if ! kubectl get deploy metrics-server -n kube-system >/dev/null 2>&1; then
        kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
        kubectl patch deploy metrics-server -n kube-system --type=json -p \
          '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
      fi
      kubectl -n kube-system rollout status deploy/metrics-server --timeout=180s || true

      # Target workload WITH cpu requests (HPA needs a request to compute % against).
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: php-apache
      spec:
        replicas: 1
        selector:
          matchLabels: {app: php-apache}
        template:
          metadata:
            labels: {app: php-apache}
          spec:
            containers:
              - name: php-apache
                image: registry.k8s.io/hpa-example
                ports: [{containerPort: 80}]
                resources:
                  requests: {cpu: 200m}
                  limits: {cpu: 500m}
      YAML
      kubectl -n "$NS" rollout status deploy/php-apache --timeout=120s

  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      kubectl -n "$NS" get hpa php-apache >/dev/null 2>&1 || {
        echo "not yet: no HPA named 'php-apache' in $NS"; exit 1; }
      ref=$(kubectl -n "$NS" get hpa php-apache -o jsonpath='{.spec.scaleTargetRef.name}')
      minr=$(kubectl -n "$NS" get hpa php-apache -o jsonpath='{.spec.minReplicas}')
      maxr=$(kubectl -n "$NS" get hpa php-apache -o jsonpath='{.spec.maxReplicas}')
      [ "$ref" = "php-apache" ] || { echo "not yet: HPA must target Deployment php-apache (got '$ref')"; exit 1; }
      [ "${minr:-0}" = "1" ] || { echo "not yet: minReplicas must be 1 (got ${minr:-unset})"; exit 1; }
      [ "${maxr:-0}" = "5" ] || { echo "not yet: maxReplicas must be 5 (got ${maxr:-unset})"; exit 1; }
      echo "PASS — HPA php-apache scales the Deployment between $minr and $maxr replicas on CPU."
---

## The situation

The `php-apache` Deployment in `kubelings` runs a single replica and falls over
under load. metrics-server is already collecting CPU usage. You need a
**HorizontalPodAutoscaler** so the app scales out on CPU and back in when idle.

## Your task

Create an HPA named **`php-apache`** that:

1. Targets the `php-apache` Deployment.
2. `minReplicas: 1`, `maxReplicas: 5`.
3. Scales on CPU (e.g. target 50% average utilization).

```sh
kubectl -n kubelings top pods          # metrics-server is live
kubectl -n kubelings get hpa
```

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings autoscale deploy php-apache --cpu=50% --min=1 --max=5
```

(Generate a load and watch with `kubectl -n kubelings get hpa -w` to see it climb.)

</details>
