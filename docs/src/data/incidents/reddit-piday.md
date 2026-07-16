---
title: "Reddit: the Pi-Day outage"
description: "[REAL] Mar 2023 — a 1.24 upgrade removed the deprecated 'master' node label; Calico route reflectors selected nodes by it, and the pod network collapsed. Kubelings capstone reading."
---

> **[REAL] incident** — cited from Reddit's public postmortem:
> ["You Broke Reddit: The Pi-Day Outage"](https://www.reddit.com/r/RedditEng/comments/11xx5o0/you_broke_reddit_the_piday_outage/).
> **Kubelings capstone reading:** lesson `incident-reddit-piday` (Module 9 — War Stories).

## Situation

14 March 2023. Reddit's infra team starts a routine, well-rehearsed Kubernetes
upgrade (1.23 → 1.24) on their oldest, largest cluster. Minutes in, the site
goes down — and stays degraded for about **five hours** (314 minutes, on Pi
Day), until the team restores the cluster from backup.

## Blast radius

- Cluster-wide pod-network collapse: pods couldn't reach pods.
- Recovery required a full **etcd restore** — under pressure, on a procedure
  never exercised at this scale, with cert/config mismatches fighting back.

## Root cause chain

1. **A deprecation landed** — Kubernetes 1.24 completed the long-announced
   rename of the control-plane node label
   (`node-role.kubernetes.io/master` → `control-plane`). Upgraded nodes simply
   stopped carrying the old label.
2. **A selector went empty** — this cluster's Calico setup chose its BGP
   **route reflectors** by node selector matching the **old label**. Label
   gone → zero route reflectors → BGP mesh collapse → no node knows how to
   route pod traffic.
3. **The snowflake tax** — the cluster predated Reddit's standardized build;
   its Calico config was hand-crafted years earlier and unique to it. Newer
   clusters didn't have the landmine; nothing rehearses a snowflake but itself.
4. **Recovery was its own incident** — downgrades are unsupported, so the path
   back was an untested etcd restore. Untested restores are recovery
   *hypotheses*.

## Fix & prevention

- **Read deprecation notes as a diff against your own configs** — a grep of
  addon configs for the dying label would have found this in seconds.
- **Kill snowflakes**: one templated, versioned cluster build; drift from the
  standard is a bug.
- **Practice the restore**, including certs and addon config, on a real
  cluster — before the day it's the only option.
- Synthetic **pod-to-pod dataplane checks**: node Ready and control-plane
  health stayed green throughout; the layer that died had no monitor.

## What it teaches

| Concept | Kubelings module |
|---|---|
| the full cascade (guided study) | M9 — `incident-reddit-piday` (reading) |
| labels & selectors as contracts | M1 — `selector-mismatch` |
| upgrade runbook & version skew | M8 — `upgrade-runbook` |
| etcd restore mechanics | M7 — `etcd-backup-restore` |
