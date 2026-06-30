---
kind: skill-path

title: "Kubelings: CKA Workloads Track"
description: |
  Hands-on CKA Workloads & Scheduling practice the rustlings way — fix, build, and
  debug broken-on-purpose Kubernetes scenarios until an automated check passes.
  Covers Deployments & rolling updates, DaemonSets, StatefulSets, Jobs/CronJobs,
  HorizontalPodAutoscaler, and right-sizing resources to stop OOMKills.

categories:
- kubernetes

tagz:
- cka
- workloads
- certification

createdAt: 2026-06-30

challenges:
  kb-wl-01-53e1821a: {}
  kb-wl-02-6c8af3fb: {}
  kb-wl-03-e73bdf82: {}
  kb-wl-04-a6bb83fd: {}
  kb-wl-05-723804ee: {}
  kb-wl-06-6c1df5e8: {}
  kb-wl-07-d4d9a2d1: {}
---

## Kubelings — CKA Workloads Track

Seven self-contained challenges on a live multi-node Kubernetes cluster. Each one
boots a broken or empty scenario and an automated check that only goes green when
you've genuinely fixed it.

1. **Fix the Rolling Update** — unsafe `maxSurge`/`maxUnavailable` causing outages.
2. **Node-Level Log Collector** — build a DaemonSet, one pod per node.
3. **StatefulSet + Headless Service** — stable pod identity and per-pod DNS.
4. **The Job That Never Finishes** — make a one-shot Job actually complete.
5. **CronJob Pileup** — stop overlapping runs with `concurrencyPolicy`.
6. **Autoscale with an HPA** — scale a Deployment 1→5 on CPU.
7. **OOMKilled CrashLoop** — right-size memory requests/limits.

Work them top to bottom; each builds intuition for the CKA *Workloads & Scheduling*
domain.
