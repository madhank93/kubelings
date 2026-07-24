---
kind: unit
title: "Incident replay — the priority that ate production (Grafana Labs)"
name: incident-priority-preemption-unit
---


## The real incident

**Grafana Labs**, July 2019. They rolled out Pod Priorities — a sensible
hardening step — and caused the very outage the feature exists to prevent.

Source: [How a production outage was caused using Kubernetes pod priorities](https://grafana.com/blog/2019/07/24/how-a-production-outage-was-caused-using-kubernetes-pod-priorities/)

The mechanism everyone underestimates: **priority isn't just a queue order —
it's an eviction license.** When a high-priority pod can't schedule, the
scheduler doesn't merely wait; it looks for nodes where **killing lower-priority
pods** would make room, and kills them. That's *preemption*, and it's on by
default for every PriorityClass.

At Grafana, priorities were introduced gradually — meaning during the migration
some production workloads had *no* priority class (implicit priority 0) while
other things had positive values. Under the next resource squeeze, Kubernetes
did exactly what it was configured to do: it evicted the "least important" pods,
which were, in fact, production. The feature worked. The configuration lied
about what was important.

## This cluster, right now

Someone copy-pasted the tiers and swapped the numbers:

```sh
kubectl get priorityclass tier-critical tier-batch
```

```
NAME            VALUE    ...
tier-critical   1000
tier-batch      100000   ← reindexing jobs outrank checkout
```

Nothing is broken *right now* — the cluster has room. That's the Grafana lesson
in miniature: **priority misconfiguration is invisible until the first resource
fight**, and then it decides who dies.

## Your task

Disarm the trap before it springs:

1. `tier-critical` must outrank `tier-batch` (PriorityClass `value` is immutable
   — you'll need to recreate, not patch).
2. Batch should never evict anyone: set its `preemptionPolicy: Never` — batch
   waits its turn.
3. Both deployments remain Available.

```sh
kubectl get priorityclass
kubectl -n kubelings get pods -o custom-columns=NAME:.metadata.name,PRIO:.spec.priority
```

<details>
<summary>Hint</summary>

Value is immutable → delete and recreate both:

```sh
kubectl delete priorityclass tier-critical tier-batch
kubectl apply -f - <<'EOF'
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata: {name: tier-critical}
value: 100000
preemptionPolicy: PreemptLowerPriority
description: "revenue-critical services"
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata: {name: tier-batch}
value: 1000
preemptionPolicy: Never
description: "batch — waits, never evicts"
EOF
```

Existing pods keep their old resolved priority until re-rolled — recreate the
classes, then `kubectl -n kubelings rollout restart deploy checkout reindex`.

</details>

::simple-task
---
:tasks: tasks
:name: verify_done
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>


## What preemption actually does

High-priority pod Pending → scheduler simulates: *"which node could fit it if I
evicted cheaper pods?"* → victims get graceful deletion (PDBs are **best-effort
only** here — priority overrules budgets!) → pod schedules into the hole.

Read that parenthetical again: **preemption can violate PodDisruptionBudgets.**
The two safety systems you've learned compose with priority *winning*. A wrong
priority number defeats every carefully-set PDB in the cluster.

## The fix, generalized (Grafana's own conclusions)

- **Order the tiers truthfully** — the numbers encode "who dies first" in kernel
  terms; treat them like production code, reviewed by people who know what's
  actually critical.
- **`preemptionPolicy: Never` for batch/optional tiers** — they queue instead of
  evict. Preemption stays reserved for genuinely critical rescue.
- **No gaps during migration:** while *anything* has priority > 0, everything
  important must too — implicit 0 is the bottom of the food chain. Use
  `globalDefault: true` on a sane middle tier so unlabeled pods don't silently
  become prey.

## Priority ≠ QoS (they compose, differently)

| | decided by | acts at |
|---|---|---|
| **QoS** (M2) | requests/limits shape | node memory pressure — kubelet eviction |
| **Priority** | PriorityClass value | scheduling time — scheduler preemption |

A Guaranteed pod with low priority survives OOM pressure but loses scheduling
fights. A BestEffort pod with high priority wins scheduling then dies first at
OOM. Set both on purpose.

## Prevention

- Grep your fleet: pods with `priorityClassName` unset in namespaces where any
  class exists = future victims list.
- Alert on `Preempted` events — each one is the scheduler killing something; you
  want to *know*.

</details>
