---
kind: lesson
title: 'Audit the cluster: CIS benchmark with kube-bench'
description: |
  You hardened pods; now audit the platform itself. Run kube-bench — the CIS
  Kubernetes Benchmark as a Job — against this cluster, read its PASS/FAIL/WARN
  verdicts on the API server, kubelet, and etcd configuration, and learn which
  findings matter before an auditor (or an attacker) reads them to you.
name: cis-kube-bench
slug: cis-kube-bench
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
      kubectl -n "$NS" delete job kube-bench --ignore-not-found
      # Nothing to break here — the "broken thing" is the cluster's own config,
      # and the task is to measure it.
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      if ! kubectl -n "$NS" get job kube-bench >/dev/null 2>&1; then
        echo "not yet: no 'kube-bench' job in the kubelings namespace — create it from the manifest in the lesson"; exit 1
      fi
      done=$(kubectl -n "$NS" get job kube-bench -o jsonpath='{.status.succeeded}' 2>/dev/null)
      if [ "${done:-0}" -lt 1 ]; then
        echo "not yet: the kube-bench job hasn't completed — kubectl -n kubelings get pods -l job-name=kube-bench (image pull can take a minute)"; exit 1
      fi
      if ! kubectl -n "$NS" logs job/kube-bench 2>/dev/null | grep -q '== Summary'; then
        echo "not yet: job finished but no benchmark summary in its logs — did it run against the node's config dirs (check the volume mounts)?"; exit 1
      fi
      echo "PASS — benchmark ran; now the important part is in the logs: kubectl -n kubelings logs job/kube-bench | less. Read the FAILs."
---
