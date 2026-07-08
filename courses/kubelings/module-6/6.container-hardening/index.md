---
kind: lesson
title: 'Hardening: take away everything the workload doesn''t use'
description: |
  The worker runs as root with a writable filesystem, every Linux capability,
  and no syscall filter — and uses none of it. Strip it to what it actually
  needs: non-root, read-only root filesystem, zero capabilities, RuntimeDefault
  seccomp. The pod should barely notice; an attacker inside it definitely will.
name: container-hardening
slug: container-hardening
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
      kubectl -n "$NS" delete deploy worker --ignore-not-found
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: worker
      spec:
        replicas: 1
        selector:
          matchLabels: {app: worker}
        template:
          metadata:
            labels: {app: worker}
          spec:
            containers:
              - name: worker
                image: busybox:1.36
                # Writes a heartbeat to /work — its ONLY filesystem need.
                command:
                  - sh
                  - -c
                  - 'while true; do date > /work/heartbeat 2>/dev/null || date > /tmp/heartbeat; sleep 5; done'
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/worker --timeout=120s
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      tpl='{.spec.template.spec}'
      nonroot=$(kubectl -n "$NS" get deploy worker -o jsonpath='{.spec.template.spec.containers[0].securityContext.runAsNonRoot}' 2>/dev/null)
      psnonroot=$(kubectl -n "$NS" get deploy worker -o jsonpath='{.spec.template.spec.securityContext.runAsNonRoot}' 2>/dev/null)
      if [ "$nonroot" != "true" ] && [ "$psnonroot" != "true" ]; then
        echo "not yet: worker may still run as root — set runAsNonRoot: true (and a runAsUser) in the securityContext"; exit 1
      fi
      rofs=$(kubectl -n "$NS" get deploy worker -o jsonpath='{.spec.template.spec.containers[0].securityContext.readOnlyRootFilesystem}' 2>/dev/null)
      if [ "$rofs" != "true" ]; then
        echo "not yet: root filesystem is still writable — set readOnlyRootFilesystem: true (and mount an emptyDir where it writes)"; exit 1
      fi
      caps=$(kubectl -n "$NS" get deploy worker -o jsonpath='{.spec.template.spec.containers[0].securityContext.capabilities.drop[0]}' 2>/dev/null)
      if [ "$caps" != "ALL" ]; then
        echo "not yet: capabilities aren't dropped — capabilities: {drop: [\"ALL\"]}"; exit 1
      fi
      sec=$(kubectl -n "$NS" get deploy worker -o jsonpath='{.spec.template.spec.containers[0].securityContext.seccompProfile.type}' 2>/dev/null)
      psec=$(kubectl -n "$NS" get deploy worker -o jsonpath='{.spec.template.spec.securityContext.seccompProfile.type}' 2>/dev/null)
      if [ "$sec" != "RuntimeDefault" ] && [ "$psec" != "RuntimeDefault" ]; then
        echo "not yet: no seccomp profile — set seccompProfile: {type: RuntimeDefault}"; exit 1
      fi
      priv=$(kubectl -n "$NS" get deploy worker -o jsonpath='{.spec.template.spec.containers[0].securityContext.allowPrivilegeEscalation}' 2>/dev/null)
      if [ "$priv" != "false" ]; then
        echo "not yet: allowPrivilegeEscalation isn't false — without it, setuid binaries can regain what you dropped"; exit 1
      fi
      avail=$(kubectl -n "$NS" get deploy worker -o jsonpath='{.status.availableReplicas}')
      if [ "${avail:-0}" -lt 1 ]; then
        echo "not yet: worker isn't running — hardening that breaks the workload isn't done yet (writable path? user id?)"; exit 1
      fi
      echo "PASS — non-root, read-only, zero capabilities, seccomp on, still working. Least privilege, demonstrated."
---
