## The situation

The session store scaled to 6 replicas last month. Check where they actually
live:

```sh
kubectl -n kubelings get pods -l app=sessions -o wide
```

Something like 5-on-one-node, 1-elsewhere. Not one node (you fixed that pattern
in the Moonlight lesson) — but nowhere near balanced. Lose the heavy node and
you lose ~83% of session capacity in one blink; the survivors take 6× their
normal load and probably OOM in the aftershock. Partial stacking is the subtler
cousin of full stacking, and anti-affinity **cannot express the fix**: it only
knows "together / not together". With 6 replicas and 2 workers, *some*
co-location is mandatory — the question is *how much*, and that's a counting
question.

`topologySpreadConstraints` is the counting tool:

```yaml
topologySpreadConstraints:
  - maxSkew: 1                          # |domainMax - domainMin| ≤ 1
    topologyKey: kubernetes.io/hostname # what counts as a domain
    whenUnsatisfiable: ScheduleAnyway   # soft (score) vs DoNotSchedule (filter)
    labelSelector:
      matchLabels: {app: sessions}
```

## Your task

1. Declare a spread constraint on `sessions` (hostname domain, maxSkew 1).
2. Re-roll so existing pods re-place under the new rule (`…IgnoredDuring
   Execution` semantics: constraints bind at scheduling time only).
3. End state: 6/6 Available, per-node counts within skew ≤ 2, constraint
   present in the spec (the check requires the *policy*, not lucky placement).

```sh
kubectl -n kubelings get pods -l app=sessions -o wide | awk '{print $7}' | sort | uniq -c
```
