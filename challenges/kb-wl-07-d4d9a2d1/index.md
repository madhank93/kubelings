---
kind: challenge

title: "OOMKilled CrashLoop: right-size the memory limit"
description: |
  A cache Deployment is stuck in CrashLoopBackOff — every pod is OOMKilled
  seconds after start. Diagnose the resource misconfiguration and give the
  workload enough memory to run.

categories:
- kubernetes

tagz:
- cka
- ckad
- workloads
- resources

difficulty: medium

createdAt: 2026-06-30

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
      # Workload needs ~50Mi but its limit is 20Mi -> OOMKilled on startup.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: cache
      spec:
        replicas: 1
        selector:
          matchLabels: {app: cache}
        template:
          metadata:
            labels: {app: cache}
          spec:
            containers:
              - name: cache
                image: polinux/stress
                command: ["stress"]
                args: ["--vm","1","--vm-bytes","50M","--vm-hang","1","-t","3600"]
                resources:
                  requests: {memory: "16Mi"}
                  limits:   {memory: "20Mi"}   # BUG: far below the ~50Mi needed
      YAML
      # Don't wait for rollout — it will be crashlooping by design.
      sleep 5 || true

  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      desired=$(kubectl -n "$NS" get deploy cache -o jsonpath='{.spec.replicas}' 2>/dev/null)
      avail=$(kubectl -n "$NS" get deploy cache -o jsonpath='{.status.availableReplicas}' 2>/dev/null)
      if [ -z "$desired" ] || [ "${avail:-0}" -lt "$desired" ]; then
        echo "not yet: cache Deployment not Available (${avail:-0}/${desired:-?}) — still OOMKilled?"; exit 1
      fi
      lim=$(kubectl -n "$NS" get pods -l app=cache \
        -o jsonpath='{range .items[*]}{.spec.containers[0].resources.limits.memory}{"\n"}{end}' 2>/dev/null)
      if [ -z "$lim" ] || grep -qx "" <<<"$lim"; then
        echo "not yet: cache pods must declare a memory limit"; exit 1
      fi
      restarts=$(kubectl -n "$NS" get pods -l app=cache \
        -o jsonpath='{range .items[*]}{.status.containerStatuses[*].restartCount}{"\n"}{end}' 2>/dev/null \
        | awk '{s+=$1} END{print s+0}')
      if [ "${restarts:-0}" -gt 2 ]; then
        echo "not yet: cache pods are still restarting (total=$restarts) — likely still OOM"; exit 1
      fi
      echo "PASS — cache is running steadily with a memory limit large enough for the workload."
---

## The situation

The `cache` Deployment in `kubelings` never stays up — pods start, then die with
**OOMKilled** and fall into CrashLoopBackOff. The container needs roughly 50Mi of
working memory, but its `limits.memory` is set to **20Mi**, so the kernel kills it
the moment it allocates.

## Your task

Right-size the memory so `cache` runs steadily:

1. Inspect why the pod is being killed.
2. Raise the memory `requests`/`limits` to fit the ~50Mi workload (give headroom).
3. The Deployment must become Available and stop restarting.

```sh
kubectl -n kubelings get pods -l app=cache
kubectl -n kubelings describe pod -l app=cache | grep -A3 -i 'last state\|reason'
```

<details>
<summary>Hint</summary>

`Reason: OOMKilled` confirms the memory cap. Raise it:

```sh
kubectl -n kubelings set resources deploy/cache \
  --requests=memory=64Mi --limits=memory=128Mi
```

</details>
