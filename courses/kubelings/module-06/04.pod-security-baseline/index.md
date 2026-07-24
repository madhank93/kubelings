---
kind: lesson
title: 'Pod Security: the privileged pod that shouldn''t exist'
description: |
  A vendor Helm chart wants a privileged, host-network pod with hostPath to the
  Docker socket — a full container escape, one YAML away from root on the node.
  Enforce Pod Security Standards on the namespace and rework the pod to run
  unprivileged.
name: pod-security-baseline
slug: pod-security-baseline
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
      # Ensure no enforcement yet (learner adds it), and clear any prior labels.
      kubectl label namespace "$NS" \
        pod-security.kubernetes.io/enforce- \
        pod-security.kubernetes.io/enforce-version- 2>/dev/null || true
      # The dangerous pod the vendor chart wants:
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: Pod
      metadata:
        name: vendor-agent
        labels: {app: vendor-agent}
      spec:
        hostNetwork: true
        containers:
          - name: agent
            image: busybox:1.36
            command: ["sh", "-c", "echo vendor agent running; while true; do sleep 10; done"]
            securityContext:
              privileged: true
              runAsUser: 0
            volumeMounts:
              - {name: dockersock, mountPath: /var/run/docker.sock}
            resources:
              requests: {cpu: 10m, memory: 16Mi}
              limits: {memory: 64Mi}
        volumes:
          - name: dockersock
            hostPath: {path: /var/run/docker.sock}
      YAML
      sleep 3 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      # 1) The namespace must ENFORCE at least the baseline standard.
      enf=$(kubectl get ns "$NS" -o jsonpath='{.metadata.labels.pod-security\.kubernetes\.io/enforce}' 2>/dev/null)
      case "$enf" in
        baseline|restricted) : ;;
        *) echo "not yet: namespace $NS does not enforce a Pod Security Standard (need baseline or restricted)"; exit 1;;
      esac
      # 2) A privileged / hostNetwork pod must no longer exist or be admissible.
      pod=$(kubectl -n "$NS" get pod vendor-agent -o jsonpath='{.spec.containers[0].securityContext.privileged}' 2>/dev/null || true)
      hnet=$(kubectl -n "$NS" get pod vendor-agent -o jsonpath='{.spec.hostNetwork}' 2>/dev/null || true)
      if [ "$pod" = "true" ] || [ "$hnet" = "true" ]; then
        echo "not yet: the privileged/hostNetwork vendor-agent pod is still present — rework it to run unprivileged"; exit 1
      fi
      # 3) Prove enforcement is real: a fresh privileged pod must be REJECTED.
      if kubectl -n "$NS" run psa-probe --image=busybox:1.36 --restart=Never \
          --overrides='{"spec":{"containers":[{"name":"c","image":"busybox:1.36","securityContext":{"privileged":true},"command":["sh","-c","sleep 1"]}]}}' \
          >/dev/null 2>&1; then
        kubectl -n "$NS" delete pod psa-probe --force --grace-period=0 >/dev/null 2>&1 || true
        echo "not yet: a privileged pod was still admitted — enforcement isn't actually blocking"; exit 1
      fi
      echo "PASS — namespace enforces the baseline and a privileged pod is rejected at admission. Policy > vigilance."
---
