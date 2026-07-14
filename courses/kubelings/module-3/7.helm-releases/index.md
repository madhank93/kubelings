---
kind: lesson
title: 'Helm: history, rollback, and the failed release'
description: |
  A Helm upgrade shipped a bad image tag and the release is now `failed` —
  pods stuck in ImagePullBackOff. Read the release history, roll back to the
  last good revision, then re-ship the intended change with corrected values.
  helm install/upgrade/rollback, the full loop.
name: helm-releases
slug: helm-releases
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
      # helm is not preinstalled in this shell — pin and verify it ourselves.
      HELM_VERSION=v3.16.4
      if ! command -v helm >/dev/null 2>&1; then
        os=$(uname -s | tr '[:upper:]' '[:lower:]')
        arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64|arm64) arch=arm64;; esac
        curl -fsSL "https://get.helm.sh/helm-${HELM_VERSION}-${os}-${arch}.tar.gz" -o /tmp/helm.tgz
        curl -fsSL "https://get.helm.sh/helm-${HELM_VERSION}-${os}-${arch}.tar.gz.sha256sum" \
          | awk '{print $1"  /tmp/helm.tgz"}' \
          | { command -v sha256sum >/dev/null 2>&1 && sha256sum -c - || shasum -a 256 -c -; }
        tar xzf /tmp/helm.tgz -C /tmp
        install "/tmp/${os}-${arch}/helm" /usr/local/bin/helm
      fi
      # Vendor a tiny chart on disk — no chart-repo network dependency.
      CHART=/tmp/kubelings-charts/orders
      mkdir -p "$CHART/templates"
      cat > "$CHART/Chart.yaml" <<'EOF'
      apiVersion: v2
      name: orders
      description: Tiny demo service for the kubelings helm lesson
      type: application
      version: 0.1.0
      appVersion: "1.0"
      EOF
      cat > "$CHART/values.yaml" <<'EOF'
      replicaCount: 1
      image:
        repository: busybox
        tag: "1.36"
      EOF
      cat > "$CHART/templates/deployment.yaml" <<'EOF'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: {{ .Release.Name }}-web
        labels:
          app: {{ .Release.Name }}-web
      spec:
        replicas: {{ .Values.replicaCount }}
        selector:
          matchLabels:
            app: {{ .Release.Name }}-web
        template:
          metadata:
            labels:
              app: {{ .Release.Name }}-web
          spec:
            containers:
              - name: web
                image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
                command: ["sh", "-c", "while true; do sleep 30; done"]
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
      EOF
      # Revision 1: healthy.
      helm uninstall orders -n "$NS" >/dev/null 2>&1 || true
      helm install orders "$CHART" -n "$NS" --wait --timeout 90s
      # Revision 2: the intended change was replicas 1->2 and a new image tag
      # — but the tag doesn't exist. --wait times out, release marked failed.
      helm upgrade orders "$CHART" -n "$NS" \
        --set replicaCount=2 --set image.tag=2.0-rc \
        --wait --timeout 45s || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! command -v helm >/dev/null 2>&1; then
        echo "not yet: helm not on PATH — re-run init"; exit 1
      fi
      if ! helm status orders -n "$NS" 2>/dev/null | grep -q "STATUS: deployed"; then
        echo "not yet: release 'orders' is not in deployed state — check 'helm history orders -n $NS'"; exit 1
      fi
      img=$(kubectl -n "$NS" get deploy orders-web -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null)
      if [ "$img" = "busybox:2.0-rc" ]; then
        echo "not yet: orders-web still runs the broken tag 2.0-rc"; exit 1
      fi
      reps=$(kubectl -n "$NS" get deploy orders-web -o jsonpath='{.spec.replicas}' 2>/dev/null)
      if [ "${reps:-0}" -ne 2 ]; then
        echo "not yet: orders-web has ${reps:-0} replicas — rollback alone cancels the release; re-ship the intended change (2 replicas, working tag)"; exit 1
      fi
      ready=$(kubectl -n "$NS" get deploy orders-web -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
      if [ "${ready:-0}" -ne 2 ]; then
        echo "not yet: only ${ready:-0}/2 pods Ready"; exit 1
      fi
      echo "PASS — rolled back the failed release, then re-shipped the change with a working tag. helm history tells the whole story."
---
