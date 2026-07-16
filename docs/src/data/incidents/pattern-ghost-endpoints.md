---
title: "Pattern: ghost endpoints"
description: "[PATTERN] Synthetic composite — pods die faster than the endpoint list updates, so every scale-down and rollout throws a burst of 503s at dead IPs."
---

> **[PATTERN] scenario** — a synthetic composite of a failure mode reported
> across many production clusters. **No specific company**; details are
> representative, not cited. (Real, cited incidents are marked `[REAL]` in the
> [Incident Library](/catalog/).)

## Situation

Every deploy and every scale-down of a service produces a short burst of 503s
— gone before anyone can look. Retries paper over it for months, until a
large scale-down turns the blip into a visible error-rate spike and an
incident review asks why "zero-downtime deploys" aren't.

## Root cause

Pod termination and endpoint removal run **in parallel, unsequenced**. On
delete, the kubelet starts killing the container (preStop → SIGTERM → grace
period → SIGKILL) while, independently, the endpoints controller removes the
pod IP and every node's kube-proxy eventually applies the update. If the
process dies before the routing mesh catches up, traffic is routed to a dead
IP: a **ghost endpoint**.

The pattern's aggravators: `terminationGracePeriodSeconds: 0` (set "to speed
up deploys"), no `preStop` hook, or an app that ignores SIGTERM and gets
SIGKILLed at the grace-period deadline mid-request.

## Diagnosis

```sh
kubectl get deploy <name> -o jsonpath='{.spec.template.spec.terminationGracePeriodSeconds}'   # 0?
kubectl get deploy <name> -o jsonpath='{.spec.template.spec.containers[0].lifecycle}'          # empty?
# reproduce: watch endpoints while deleting a pod —
kubectl get endpoints <svc> -w
```

Client-side symptom: 502/503s correlated exactly with deploys and
scale-downs, never in steady state.

## Fix

```yaml
spec:
  template:
    spec:
      terminationGracePeriodSeconds: 30
      containers:
        - name: app
          lifecycle:
            preStop:
              exec:
                command: ["sh", "-c", "sleep 5"]
```

The preStop sleep deliberately holds the pod alive and serving while its IP
leaves every node's rules; only then does SIGTERM begin the app's own
shutdown (stop accepting, drain in-flight, exit).

## Prevention

- Termination budget: `grace period ≥ preStop sleep + app shutdown time`.
- The app must handle SIGTERM — a process that needs SIGKILL turns every
  rollout into small data loss.
- Test it: `kubectl delete pod` under load in staging; watch error rate.
- Grace period 0 belongs to `kubectl delete --force` emergencies only, never
  in a template.

## What it teaches

| Concept | Kubelings module |
|---|---|
| Termination lifecycle, preStop, graceful shutdown | M3 (`pattern-ghost-endpoints`) |
| Endpoint propagation & kube-proxy dataplane | M4 Networking (`kube-proxy-dataplane`) |
