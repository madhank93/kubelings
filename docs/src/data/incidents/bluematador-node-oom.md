---
title: "Blue Matador: when the node runs out, the kernel chooses"
description: "[REAL] — pods without memory limits grew until the node itself OOMed; the kernel killer shot processes at random, including system daemons. Runnable Kubelings replay."
---

> **[REAL] incident** — cited from Blue Matador's public postmortem:
> [Kubernetes node OOM post-mortem](https://www.bluematador.com/blog/post-mortem-kubernetes-node-oom).
> **Runnable replay:** Kubelings lesson `incident-node-oom` (Module 8 — Observability & SRE).

## Situation

Blue Matador (a monitoring company — the irony is a genre convention at this
point) hit node instability: workloads dying, nodes flapping, and kills that
didn't match any Kubernetes-visible event.

## Blast radius

- Processes killed by the **kernel's** OOM killer, not the kubelet — victims
  chosen by kernel heuristics, so system daemons were fair game.
- Node-level instability: when node components die, *every* pod on the node
  inherits the outage.

## Root cause chain

1. **The gap** — pods ran **without memory limits**. Kubernetes scheduled them
   by requests (or none), but nothing bounded their growth.
2. **Two killers, different rules** — a pod exceeding its own limit gets a
   clean, attributable **OOMKilled** (restart, `kubectl describe` shows it —
   the M2 `oomkill` lesson). But if the *node* runs out first, the **kernel**
   OOM killer picks victims by oom_score across all processes.
3. **The escalation path** — kubelet eviction (`memory.available` threshold)
   is *supposed* to fire first and evict by QoS class — but memory spikes can
   outrun the kubelet's sampling, jumping straight to the kernel.
4. **The attribution hole** — kernel kills don't produce tidy pod events;
   you're reading `dmesg` and node logs to find out what happened.

## Fix & prevention

- **Memory limits on everything** — a pod that dies at its own limit is a
  contained, attributable failure; a pod with no limit exports its failure to
  the node (QoS: BestEffort/Burstable die first, `qos-classes`).
- LimitRange per namespace so "forgot to set limits" is impossible
  (pairs with ResourceQuota, `quota-exhausted`).
- Kubelet eviction headroom: reserve memory for system daemons
  (`kube-reserved`/`system-reserved`) so the kubelet wins the race against the
  kernel.
- Alert on node `MemoryPressure` and on containers running >90% of their limit.

## What it teaches

| Concept | Kubelings module |
|---|---|
| node OOM vs pod OOM | M8 — `incident-node-oom` (runnable) |
| right-sizing memory | M2 — `oomkill` |
| QoS eviction order | M2 — `qos-classes` |
