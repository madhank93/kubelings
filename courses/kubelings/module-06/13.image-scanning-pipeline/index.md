---
kind: lesson
title: 'Scan it, then pin it: trivy and the digest'
description: |
  A Deployment is running nginx:1.14 — an image with a museum's worth of
  known CVEs — referenced by a mutable tag on top. Scan it with trivy, find
  a clean replacement, and pin the fix by digest so nobody can swap the
  bytes behind your back.
name: image-scanning-pipeline
slug: image-scanning-pipeline
createdAt: "2026-07-13"
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
      # trivy is not preinstalled — pin and install it.
      TRIVY_VERSION=0.72.0
      if ! command -v trivy >/dev/null 2>&1; then
        os=$(uname -s); arch=$(uname -m)
        case "$os" in Linux) os=Linux;; Darwin) os=macOS;; esac
        case "$arch" in x86_64) arch=64bit;; aarch64|arm64) arch=ARM64;; esac
        curl -fsSL "https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}/trivy_${TRIVY_VERSION}_${os}-${arch}.tar.gz" -o /tmp/trivy.tgz
        tar xzf /tmp/trivy.tgz -C /tmp trivy
        install /tmp/trivy /usr/local/bin/trivy
      fi
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: legacy-api
        labels: {app: legacy-api}
      spec:
        replicas: 1
        selector:
          matchLabels: {app: legacy-api}
        template:
          metadata:
            labels: {app: legacy-api}
          spec:
            containers:
              - name: legacy-api
                # 2018 called; it wants its CVEs back. Mutable tag, too.
                image: nginx:1.14
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits: {memory: 128Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/legacy-api --timeout=180s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    timeout_seconds: 300
    run: |
      NS=kubelings
      img=$(kubectl -n "$NS" get deploy legacy-api -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null)
      if [ -z "$img" ]; then
        echo "not yet: deployment legacy-api not found"; exit 1
      fi
      case "$img" in
        *@sha256:*) : ;;
        *) echo "not yet: image is '$img' — still a mutable tag; pin it by @sha256: digest"; exit 1 ;;
      esac
      if ! kubectl -n "$NS" rollout status deploy/legacy-api --timeout=120s >/dev/null 2>&1; then
        echo "not yet: legacy-api rollout not complete — does the pinned digest actually exist?"; exit 1
      fi
      if ! trivy image --severity CRITICAL --exit-code 1 --quiet "$img" >/dev/null 2>&1; then
        echo "not yet: the pinned image still carries CRITICAL vulnerabilities — scan candidates with trivy before pinning"; exit 1
      fi
      echo "PASS — scanned, swapped, and pinned by digest. The image can no longer rot or be swapped behind the tag."
---
