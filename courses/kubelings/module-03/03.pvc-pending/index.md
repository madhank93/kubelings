---
kind: lesson
title: 'PVC Pending: a claim nobody answers'
description: |
  The analytics database needs a disk. Its PersistentVolumeClaim has been Pending
  for half an hour — it names a StorageClass that doesn't exist in this cluster.
  Learn the PV/PVC/StorageClass triangle and how dynamic provisioning actually
  answers a claim.
name: pvc-pending
slug: pvc-pending
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
        name: analytics-data
      spec:
        accessModes: [ReadWriteOnce]
        # BUG: 'fast-ssd' exists in the cloud clusters, not here. No provisioner answers.
        storageClassName: fast-ssd
        resources:
          requests: {storage: 1Gi}
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: analytics
      spec:
        replicas: 1
        selector:
          matchLabels: {app: analytics}
        template:
          metadata:
            labels: {app: analytics}
          spec:
            containers:
              - name: analytics
                image: busybox:1.36
                command: ["sh", "-c", "echo analytics up, data dir:; ls -ld /data; while true; do sleep 5; done"]
                volumeMounts:
                  - {name: data, mountPath: /data}
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits: {memory: 64Mi}
            volumes:
              - name: data
                persistentVolumeClaim:
                  claimName: analytics-data
      YAML
      sleep 5 || true
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      phase=$(kubectl -n "$NS" get pvc analytics-data -o jsonpath='{.status.phase}' 2>/dev/null)
      if [ "$phase" != "Bound" ]; then
        echo "not yet: PVC analytics-data is ${phase:-missing}, needs Bound — which StorageClass can actually answer it?"; exit 1
      fi
      avail=$(kubectl -n "$NS" get deploy analytics -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ "${avail:-0}" -lt 1 ]; then
        echo "not yet: analytics not Available"; exit 1
      fi
      pod=$(kubectl -n "$NS" get pods -l app=analytics -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
      if ! kubectl -n "$NS" exec "$pod" -- sh -c 'touch /data/.write-test && rm /data/.write-test' 2>/dev/null; then
        echo "not yet: /data is not writable inside the analytics pod"; exit 1
      fi
      echo "PASS — claim answered, volume bound, disk writable. PVC → StorageClass → provisioner → PV."
---
