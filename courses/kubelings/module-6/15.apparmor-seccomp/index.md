---
kind: lesson
title: 'seccomp on, AppArmor understood'
description: |
  A pod is running with an unfiltered syscall surface — all ~350 syscalls
  exposed to whatever escapes the app. Turn on the runtime's default seccomp
  profile the Kubernetes way, then read the AppArmor runbook for the
  host-side twin you'll meet on real nodes and the CKS.
name: apparmor-seccomp
slug: apparmor-seccomp
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
      kubectl -n "$NS" delete pod audit-me --ignore-not-found --wait=true
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: Pod
      metadata:
        name: audit-me
        labels: {app: audit-me}
      spec:
        # No securityContext at all: seccomp=Unconfined on most runtimes.
        containers:
          - name: audit-me
            image: busybox:1.36
            command: ["sh", "-c", "while true; do sleep 30; done"]
            resources:
              requests: {cpu: 10m, memory: 16Mi}
              limits: {memory: 64Mi}
      YAML
      kubectl -n "$NS" wait --for=condition=Ready pod/audit-me --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      phase=$(kubectl -n "$NS" get pod audit-me -o jsonpath='{.status.phase}' 2>/dev/null)
      if [ "$phase" != "Running" ]; then
        echo "not yet: pod audit-me is ${phase:-missing} — it must be Running"; exit 1
      fi
      podprof=$(kubectl -n "$NS" get pod audit-me -o jsonpath='{.spec.securityContext.seccompProfile.type}' 2>/dev/null)
      ctrprof=$(kubectl -n "$NS" get pod audit-me -o jsonpath='{.spec.containers[0].securityContext.seccompProfile.type}' 2>/dev/null)
      if [ "$podprof" != "RuntimeDefault" ] && [ "$ctrprof" != "RuntimeDefault" ]; then
        echo "not yet: audit-me has no RuntimeDefault seccompProfile (pod- or container-level) — the syscall surface is still wide open"; exit 1
      fi
      echo "PASS — RuntimeDefault seccomp active: the syscall surface just shrank from ~350 to what containers actually need."
---
