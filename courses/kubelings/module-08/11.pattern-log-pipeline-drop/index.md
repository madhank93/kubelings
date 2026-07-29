---
kind: lesson
title: 'Drill — the log pipeline that ships nothing'
description: |
  Synthetic drill of a failure pattern reported across many production clusters:
  the app logs, the shipper runs, every pod is Ready — and the log backend has
  been empty for a week. A tail input matching a path that never existed drops
  every record silently. Find the gap between "the agent is healthy" and "the
  records arrived".
name: pattern-log-pipeline-drop
slug: pattern-log-pipeline-drop
createdAt: "2026-07-27"
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
      kubectl -n "$NS" delete deploy payments --ignore-not-found
      kubectl -n "$NS" delete configmap shipper-config --ignore-not-found
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: shipper-config
      data:
        fluent-bit.conf: |
          [SERVICE]
              Flush        1
              Log_Level    info
              Daemon       Off
          [INPUT]
              Name         tail
              # BUG: the vendor app writes /var/log/app/app.log. This glob has
              # matched nothing since the day it was merged — and a tail input
              # with no files is not an error, it is just quiet.
              Path         /var/log/app/*.json
              Tag          payments.*
              Read_from_Head On
              Refresh_Interval 5
          [OUTPUT]
              Name         stdout
              Match        payments.*
              Format       json_lines
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: payments
      spec:
        replicas: 1
        selector:
          matchLabels: {app: payments}
        template:
          metadata:
            labels: {app: payments}
          spec:
            volumes:
              - name: applogs
                emptyDir: {}
              - name: config
                configMap:
                  name: shipper-config
            containers:
              # The vendor application. Its log path is not yours to choose.
              - name: app
                image: busybox:1.36
                command:
                  - sh
                  - -c
                  - |
                    mkdir -p /var/log/app
                    i=0
                    while true; do
                      i=$((i+1))
                      echo "{\"ts\":\"$(date -Iseconds)\",\"level\":\"info\",\"msg\":\"charge captured\",\"order_id\":\"ord-$i\"}" >> /var/log/app/app.log
                      sleep 2
                    done
                volumeMounts:
                  - {name: applogs, mountPath: /var/log/app}
                resources:
                  requests: {cpu: 10m, memory: 16Mi}
                  limits:   {memory: 64Mi}
              # The log shipper: healthy, running, forwarding nothing.
              - name: shipper
                image: fluent/fluent-bit:3.1.9
                args: ["-c", "/fluent-bit/etc/fluent-bit.conf"]
                volumeMounts:
                  - {name: applogs, mountPath: /var/log/app, readOnly: true}
                  - {name: config, mountPath: /fluent-bit/etc}
                resources:
                  requests: {cpu: 10m, memory: 32Mi}
                  limits:   {memory: 128Mi}
      YAML
      kubectl -n "$NS" rollout status deploy/payments --timeout=240s
      echo "payments is Ready — app + shipper. The backend has seen nothing for a week."
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      cmd=$(kubectl -n "$NS" get deploy payments -o jsonpath='{.spec.template.spec.containers[0].command}' 2>/dev/null)
      case "$cmd" in
        *app.log*) : ;;
        *) echo "not yet: the app no longer writes /var/log/app/app.log — that path is the vendor's. Fix the shipper, not the application"; exit 1 ;;
      esac
      out=$(kubectl -n "$NS" get configmap shipper-config -o jsonpath='{.data.fluent-bit\.conf}' 2>/dev/null)
      case "$out" in
        *stdout*) : ;;
        *) echo "not yet: the shipper's stdout OUTPUT is gone — keep shipping records somewhere observable"; exit 1 ;;
      esac
      # The only proof that counts: records leaving the agent.
      seen=""
      for _ in $(seq 1 24); do
        pod=$(kubectl -n "$NS" get pods -l app=payments \
                --field-selector=status.phase=Running \
                -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
        if [ -n "$pod" ]; then
          if kubectl -n "$NS" logs "$pod" -c shipper --tail=200 2>/dev/null | grep -q 'order_id'; then
            seen=yes; break
          fi
        fi
        sleep 5
      done
      if [ -z "$seen" ]; then
        echo "not yet: the shipper still forwards no records. Check what its tail INPUT matches (kubectl -n $NS exec deploy/payments -c app -- ls /var/log/app), and remember a ConfigMap edit needs a pod restart to take effect"; exit 1
      fi
      echo "PASS — records are leaving the agent. 'The agent is Ready' and 'the logs arrived' are two different checks; now you have the second one."
---
