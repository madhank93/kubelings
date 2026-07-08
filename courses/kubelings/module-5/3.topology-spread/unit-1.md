---
kind: unit
title: "Topology spread: balance, not just separation"
name: topology-spread-unit
---


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

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch deploy sessions --type=strategic -p '
spec:
  template:
    spec:
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels: {app: sessions}
'
kubectl -n kubelings rollout restart deploy/sessions
kubectl -n kubelings rollout status deploy/sessions
```

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


## How the imbalance happened (no villain required)

The original stacking came from a pin, but the pattern arises organically:
rolling updates surge onto whichever nodes have room *at that moment*; scale-ups
land on the emptiest node *right then*; node replacements shuffle everything.
Placement is a sequence of locally-greedy decisions. **Without a declared
constraint, balance is an accident that decays.**

## maxSkew math (worth being precise)

Skew = (pods in fullest domain) − (pods in emptiest *eligible* domain).

- 6 replicas, 2 workers, `maxSkew: 1` → 3+3 (or 4+2 briefly during rollouts).
- `whenUnsatisfiable: DoNotSchedule` makes it a hard filter — safe-sounding,
  but during a node outage it can refuse to schedule *anywhere* (the remaining
  node would exceed skew), converting partial degradation into full. That's why
  the fleet default is `ScheduleAnyway` + an alert on observed skew.
- The control plane doesn't count: its taint makes it ineligible, and eligible
  domains are what skew is computed over.

## vs anti-affinity, one line each

- **Anti-affinity:** binary separation. Right when co-location is *forbidden*
  (two replicas of a quorum member on one node = data loss risk).
- **Topology spread:** proportional balance. Right when replicas > domains —
  which is most real services.
- They compose: spread across zones + anti-affinity within zone is a common
  production pair.

## Prevention

- Any service with replicas > node count: spread constraint in the template,
  from day one.
- Watch actual balance, not just the spec:
  `kubectl get pods -l app=X -o wide | awk '{print $7}' | sort | uniq -c` —
  the verify check's exact logic, worth an alias.

</details>
