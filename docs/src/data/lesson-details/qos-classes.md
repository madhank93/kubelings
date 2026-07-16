## The situation

A node runs out of memory. This is not hypothetical — it happens in every fleet,
every week. The kernel must reclaim memory *now*, and something must die.

Kubernetes doesn't flip a coin. Every pod gets a **QoS class** at creation,
computed silently from its resources — and that class is the kill order:

| Class | Rule | Under pressure |
|---|---|---|
| **Guaranteed** | every container: requests == limits (cpu *and* memory) | evicted last |
| **Burstable** | at least one request set, but not Guaranteed | middle |
| **BestEffort** | no requests, no limits anywhere | **evicted first** |

Your namespace tonight:

- `pay-core` — the payment service. Has requests *and* limits, but they differ →
  **Burstable**. One tier above the chopping block is not where payments belongs.
- `batch-reindex` — an overnight job with **no resources declared at all** →
  **BestEffort**. Worse: with no requests, the scheduler thinks it's *free* and
  packs it anywhere, which is exactly how nodes end up under pressure.

```sh
kubectl -n kubelings get pods -o custom-columns=NAME:.metadata.name,QOS:.status.qosClass
```

## Your task

1. Make `pay-core` **Guaranteed**: requests == limits for cpu *and* memory.
2. Make `batch-reindex` honest: give it real requests (Burstable is fine —
   batch work *should* be sacrificed before payments).
3. Both deployments stay Available.
