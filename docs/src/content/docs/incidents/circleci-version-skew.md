---
title: "CircleCI: the upgrade that corrupted every node's iptables"
description: "[REAL] Mar 2023 — kubelet/kube-proxy version skew mid-upgrade changed iptables ruleset format; syncProxyRules failed cluster-wide, and recovery took a node-by-node restart plus two follow-on incidents."
---

> **[REAL] incident** — cited from CircleCI's official incident report:
> [2023-03-14 — Delays starting jobs](https://discuss.circleci.com/t/incident-report-2023-03-14-delays-starting-jobs/47555).
> **Related Kubelings lessons:** `upgrade-runbook` (M8), `kube-proxy-dataplane`
> (M4), `incident-reddit-piday` (M9 — the other upgrade war story).

## Situation

14 March 2023, 18:00 UTC — exactly the Pi-Day date Reddit went down, coincidentally —
CircleCI's main production cluster starts failing to run customer jobs during
a **staged Kubernetes upgrade**. Service-to-service networking degrades
across the cluster while every individual component looks upgraded correctly.

## Blast radius

- Customer pipelines delayed, then not running at all — ~7 hours to primary
  resolution.
- Per-node iptables left **corrupted**, so failures were scattered and
  inconsistent (which node your traffic hit determined whether it worked).
- Recovery required a **full node-by-node cluster restart** — and the
  turbulence triggered **two follow-on incidents** (RabbitMQ queue corruption,
  GitHub-checks delays into the next morning).

## Root cause chain

1. **The skew** — the staged upgrade left **kubelet and kube-proxy at
   incompatible versions** on nodes. Version skew has documented rules
   (see `upgrade-runbook`), and component-pairs on one node are the
   tightest coupling of all.
2. **The format change** — between those versions, the *format of
   kube-proxy's iptables rulesets changed*.
3. **The failing sync** — kube-proxy's `syncProxyRules()` is a
   read-modify-write over the whole ruleset (`iptables-save` →
   modify → `iptables-restore`; the mechanism from `kube-proxy-dataplane`).
   Reading rules written in the other version's format made the restore
   fail — `"Sync failed"`, repeatedly.
4. **The silent decay** — every pod churn and Endpoints change triggers a
   sync. Each failed sync left that node's iptables **stale or corrupted**,
   so Service routing rotted node by node as normal churn continued. Nothing
   was "down"; the dataplane was quietly diverging from cluster state.
5. **The reset** — there's no repair path for corrupt rulesets short of
   rebuilding them: restart every node, let kube-proxy write fresh rules
   from scratch.

## Fix & prevention

- **Version-skew discipline per node, not just per cluster**: kubelet and
  kube-proxy on one node upgrade together; control plane first, then whole
  nodes (the `upgrade-runbook` sequence exists precisely for this).
- **Watch the sync metrics during upgrades**:
  `kubeproxy_sync_proxy_rules_last_timestamp_seconds` going stale or sync
  error counts rising is this incident's early-warning signal — before
  customers see routing rot.
- **Canary + bake per node wave** — the corruption was churn-driven and
  cumulative, so a bake period on the first upgraded nodes shows it.
- **Expect aftershocks**: full-cluster restarts strain everything attached
  (message queues, webhooks). Their two follow-on incidents are the
  recovery-is-its-own-incident lesson from Reddit's Pi-Day, independently
  reproduced on the same calendar date.

## What it teaches

| Concept | Kubelings module |
|---|---|
| version skew rules & upgrade sequencing | M8 — `upgrade-runbook` |
| what syncProxyRules actually rewrites | M4 — `kube-proxy-dataplane` |
| upgrades as the top scheduled risk | M9 — `incident-reddit-piday` |
