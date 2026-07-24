---
kind: lesson
title: 'Drill — the PVC stuck Terminating'
description: |
  Synthetic drill of a failure pattern reported across many production clusters:
  a teardown deletes a PersistentVolumeClaim, it goes Terminating… and stays
  there. Forever. A finalizer is protecting it from a consumer nobody remembers.
  Find the ghost, release the claim — without the corrupting shortcut.
name: pattern-pvc-terminating
slug: pattern-pvc-terminating
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
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: PersistentVolumeClaim
      metadata:
        name: data-old
      spec:
        accessModes: [ReadWriteOnce]
        storageClassName: standard
        resources:
          requests: {storage: 1Gi}
      ---
      apiVersion: v1
      kind: Pod
      metadata:
        name: debug-shell-leftover
        labels: {purpose: debugging}
      spec:
        containers:
          - name: shell
            image: busybox:1.36
            command: ["sh", "-c", "while true; do sleep 30; done"]
            volumeMounts:
              - {name: old, mountPath: /mnt/old}
            resources:
              requests: {cpu: 10m, memory: 16Mi}
              limits: {memory: 64Mi}
        volumes:
          - name: old
            persistentVolumeClaim:
              claimName: data-old
      YAML
      kubectl -n "$NS" wait --for=condition=Ready pod/debug-shell-leftover --timeout=180s
      # The "teardown": delete the PVC. pvc-protection finalizer holds it Terminating.
      kubectl -n "$NS" delete pvc data-old --wait=false
      sleep 3 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if kubectl -n "$NS" get pvc data-old >/dev/null 2>&1; then
        # Distinguish honest fix from finalizer-stripping.
        fins=$(kubectl -n "$NS" get pvc data-old -o jsonpath='{.metadata.finalizers}' 2>/dev/null)
        if [ -z "$fins" ]; then
          echo "not yet: finalizers were stripped but the PVC still exists — that's the corrupting shortcut, not the fix"; exit 1
        fi
        echo "not yet: PVC data-old still Terminating — something still mounts it. Who?"; exit 1
      fi
      if kubectl -n "$NS" get pod debug-shell-leftover >/dev/null 2>&1; then
        echo "not yet: the ghost consumer is still running"; exit 1
      fi
      echo "PASS — consumer removed first, finalizer released the claim itself. Order beats force, every time."
---
