---
kind: course

title: Kubelings — Learn Kubernetes the Rustlings Way

description: |
  Learn Kubernetes by fixing small, broken-on-purpose clusters — one scenario at
  a time — until an automated check turns green. Each lesson drops you into a live
  multi-node cluster with a realistic fault to diagnose and repair: rolling
  updates, DaemonSets, StatefulSets, Jobs & CronJobs, autoscaling, and resource
  limits. Hands-on, check-driven, CKA/CKAD-aligned.

categories:
- kubernetes

tagz:
- cka
- ckad
- workloads
- hands-on

createdAt: 2026-06-30
updatedAt: 2026-06-30

cover: __static__/cover.jpg
---

**Kubelings** is hands-on Kubernetes practice in the spirit of *rustlings* and
*golings*: a sequence of broken or empty clusters, each with an automated check
that only passes when you've genuinely fixed the problem.

Every lesson runs in its own live, multi-node cluster — no setup, no cleanup.
Read the situation, fix the cluster with `kubectl`, and watch the check go green.

Work the modules top to bottom: Foundations → Workloads → Config & Storage →
Networking → Scheduling → Security → **Internals** → Observability & SRE →
**War Stories** — capstone replays of real, cited production postmortems (the
Zalando ndots DNS outage is already a runnable lesson in Networking).

You can also run every scenario locally on `kind` — see the project repo for the
local runner and the full incident library.
