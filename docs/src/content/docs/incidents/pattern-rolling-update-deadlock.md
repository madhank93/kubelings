---
title: "Pattern: rolling update deadlock"
description: "[PATTERN] Synthetic composite — maxUnavailable: 0 meets a revision that can never become Ready; the rollout waits forever and nothing pages."
---

> **[PATTERN] scenario** — a synthetic composite of a failure mode reported
> across many production clusters. **No specific company**; details are
> representative, not cited. (Real, cited incidents are marked `[REAL]` in the
> [Incident Library](/catalog/).)

## Situation

A release has been "rolling out" for six hours:

```
Waiting for deployment "inventory" rollout to finish: 1 old replicas are pending termination...
```

Nothing is crashing, nothing is down — old pods serve traffic, dashboards are
green. One new pod sits `Pending`. The team notices only when the *next*
release queues up behind this one.

## Root cause

Two individually-correct decisions that compose into a deadlock:

1. **`maxUnavailable: 0`** — the zero-downtime strategy: no old pod may
   terminate until a replacement is Ready.
2. **A new revision that can never become Ready** — an oversized resource
   request that fits no node, an unpullable image, a failing readiness probe,
   an unsatisfiable nodeSelector, a missing PVC.

The surge pod (`maxSurge: 1` — the API rejects both knobs at zero) stays
`Pending`; with zero unavailability budget, no old pod may be removed. The
Deployment controller isn't stuck on a bug — it is correctly refusing to
violate the stated availability policy, indefinitely and silently. With
`maxUnavailable: 1` the same bad revision would have *visibly* failed on the
first swapped pod.

## Diagnosis

```sh
kubectl rollout status deploy/<name>                       # stuck message
kubectl get deploy <name> -o jsonpath='{.status.conditions[?(@.type=="Progressing")]}'
# → reason: ProgressDeadlineExceeded (after progressDeadlineSeconds, default 600)
kubectl describe pod <pending-pod>                          # Events name the blocker:
# 0/6 nodes are available: insufficient cpu.
```

## Fix

Fix the **cause** in the pod template (creates a new revision, rollout
completes without ever dropping below capacity):

```sh
kubectl patch deploy <name> --type=json -p '[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests",
   "value": {"cpu": "100m", "memory": "128Mi"}}
]'
```

`kubectl rollout undo` is the right call when the new version itself is bad —
but when the release is fine and only its scheduling constraints are wrong,
undo just re-queues the same deadlock for the next attempt.

## Prevention

- `maxUnavailable: 0` is a **capacity commitment**: it requires schedulable
  headroom for `maxSurge` pods on every rollout, not just the YAML flag.
- Alert on `Progressing=False` / `ProgressDeadlineExceeded` — the exact signal
  every victim of this pattern was missing. Tune `progressDeadlineSeconds`
  below your deploy-pipeline timeout.
- CI/CD must watch rollouts to completion (`kubectl rollout status
  --timeout=…`) — fire-and-forget `kubectl apply` reports this deadlock as
  success.
- Resource requests are per-environment config; the cross-environment
  copy-paste is this pattern's most common trigger.

## What it teaches

| Concept | Kubelings module |
|---|---|
| RollingUpdate strategy mechanics, surge/unavailability budgets | M2 Workloads (`pattern-rolling-update-deadlock`) |
| Scheduling failures & capacity headroom | M5 Scheduling |
