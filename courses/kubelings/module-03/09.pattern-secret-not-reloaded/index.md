---
kind: lesson
title: 'Drill — the Secret that was rotated but never reloaded'
description: |
  Synthetic drill of a failure pattern reported across many production
  clusters: security rotated a database password, the Secret shows the new
  value, the app still authenticates with the old one — because the pod got
  it as an env var at startup, and env vars never reload. Move the injection
  to a volume mount and pick up the rotation.
name: pattern-secret-not-reloaded
slug: pattern-secret-not-reloaded
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
      kubectl -n "$NS" delete pod billing --ignore-not-found --wait=true
      # The world before rotation:
      kubectl -n "$NS" create secret generic db-creds \
        --from-literal=password='hunter2-2024Q4' \
        --dry-run=client -o yaml | kubectl apply -f -
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: Pod
      metadata:
        name: billing
        labels: {app: billing}
      spec:
        containers:
          - name: billing
            image: busybox:1.36
            # BUG: env injection — resolved once, at container start.
            command: ["sh", "-c", "while true; do echo \"connecting to db with password=$DB_PASSWORD\"; sleep 15; done"]
            env:
              - name: DB_PASSWORD
                valueFrom:
                  secretKeyRef: {name: db-creds, key: password}
            resources:
              requests: {cpu: 10m, memory: 16Mi}
              limits: {memory: 64Mi}
      YAML
      kubectl -n "$NS" wait --for=condition=Ready pod/billing --timeout=120s
      # The rotation: security replaces the password. The pod never notices.
      kubectl -n "$NS" create secret generic db-creds \
        --from-literal=password='s3cure-NEW-9917' \
        --dry-run=client -o yaml | kubectl apply -f -
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      phase=$(kubectl -n "$NS" get pod billing -o jsonpath='{.status.phase}' 2>/dev/null)
      if [ "$phase" != "Running" ]; then
        echo "not yet: pod billing is ${phase:-missing} — it must be Running"; exit 1
      fi
      vol=$(kubectl -n "$NS" get pod billing -o jsonpath='{.spec.volumes[?(@.secret.secretName=="db-creds")].name}' 2>/dev/null)
      if [ -z "$vol" ]; then
        echo "not yet: billing still gets db-creds via env — env vars never see a rotation; mount the Secret as a volume"; exit 1
      fi
      mp=$(kubectl -n "$NS" get pod billing -o jsonpath="{.spec.containers[0].volumeMounts[?(@.name=='$vol')].mountPath}" 2>/dev/null)
      if [ -z "$mp" ]; then
        echo "not yet: the db-creds volume exists but no container mounts it"; exit 1
      fi
      live=$(kubectl -n "$NS" exec billing -- cat "$mp/password" 2>/dev/null)
      if [ "$live" != "s3cure-NEW-9917" ]; then
        echo "not yet: the file at $mp/password does not hold the rotated value — kubelet syncs mounted Secrets within ~1 min; check and re-verify"; exit 1
      fi
      echo "PASS — Secret mounted as a file, rotated value live in the pod. Rotations now land without a redeploy."
---
