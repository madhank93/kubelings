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
