---
kind: lesson
title: 'VPA: the recommender that watched the wrong app'
description: |
  Right-sizing by hand (like the oomkill lesson) doesn't scale past a few
  services — the Vertical Pod Autoscaler watches real usage and recommends
  requests. This cluster's VPA has watched for days and recommended
  nothing: its targetRef names a Deployment that doesn't exist. Point it
  at the real workload and read your first recommendation.
name: vpa
slug: vpa
createdAt: "2026-07-14"
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
      # Metrics pipeline first: the recommender is blind without it.
      kubectl apply --server-side --force-conflicts -f \
        https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.8.0/components.yaml
      # Lab-only: kubelets here use self-signed serving certs.
      kubectl -n kube-system patch deploy metrics-server --type=json -p '[
        {"op": "add", "path": "/spec/template/spec/containers/0/args/-",
         "value": "--kubelet-insecure-tls"}
      ]' || true
      kubectl -n kube-system rollout status deploy/metrics-server --timeout=180s
      # VPA, pinned release manifests — recommender only (no updater, no
      # admission webhook: recommendations without any pod mutation).
      VPA_TAG=vertical-pod-autoscaler-1.7.0
      BASE="https://raw.githubusercontent.com/kubernetes/autoscaler/${VPA_TAG}/vertical-pod-autoscaler/deploy"
      kubectl apply --server-side --force-conflicts -f "$BASE/vpa-v1-crd-gen.yaml"
      kubectl apply --server-side --force-conflicts -f "$BASE/vpa-rbac.yaml"
      kubectl apply --server-side --force-conflicts -f "$BASE/recommender-deployment.yaml"
      kubectl -n kube-system rollout status deploy/vpa-recommender --timeout=180s
      # A workload with real, measurable usage: a modest CPU burner.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: billing-api
        labels: {app: billing-api}
      spec:
        replicas: 1
        selector:
          matchLabels: {app: billing-api}
        template:
          metadata:
            labels: {app: billing-api}
          spec:
            containers:
              - name: billing-api
                image: busybox:1.36
                # ~constant low CPU: spin a little, sleep a little.
                command: ["sh", "-c", "while true; do i=0; while [ $i -lt 20000 ]; do i=$((i+1)); done; sleep 0.2; done"]
                resources:
                  requests: {cpu: 500m, memory: 256Mi}
                  limits: {cpu: "1", memory: 512Mi}
      ---
      apiVersion: autoscaling.k8s.io/v1
      kind: VerticalPodAutoscaler
      metadata:
        name: billing-api
      spec:
        targetRef:
          apiVersion: apps/v1
          kind: Deployment
          # BUG: the service was renamed billing -> billing-api months ago;
          # the VPA still watches the ghost.
          name: billing
        updatePolicy:
          updateMode: "Off"
      YAML
      kubectl -n "$NS" rollout status deploy/billing-api --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    timeout_seconds: 300
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get vpa billing-api >/dev/null 2>&1; then
        echo "not yet: VPA billing-api is gone — fix its targetRef, don't delete it"; exit 1
      fi
      target=$(kubectl -n "$NS" get vpa billing-api -o jsonpath='{.spec.targetRef.name}')
      if [ "$target" != "billing-api" ]; then
        echo "not yet: VPA still targets Deployment '$target' — which Deployment actually exists here?"; exit 1
      fi
      rec=$(kubectl -n "$NS" get vpa billing-api -o jsonpath='{.status.recommendation.containerRecommendations[0].target}' 2>/dev/null)
      if [ -z "$rec" ]; then
        echo "not yet: no recommendation yet — the recommender runs every minute; check it sees the target (kubectl -n kube-system logs deploy/vpa-recommender) and re-verify shortly"; exit 1
      fi
      echo "PASS — the recommender watched real usage and wrote its verdict: $rec. Compare that to the 500m the template requests."
---
