---
kind: unit
title: "Incident replay — all replicas on the failing node (Moonlight)"
name: incident-same-node-unit
---


## The real incident

**Moonlight** (developer job marketplace, GKE). Their status page one morning:
website down, API down — **100% traffic loss**. The postmortem's punchline: every
replica of the affected service had been scheduled onto **the same node**, and
that node experienced a kernel panic. Redundant replicas, zero redundancy.

Source: [Moonlight outage post-mortem](https://updates.moonlightwork.com/outage-post-mortem-87370)

The uncomfortable truth this incident teaches: **the scheduler does not spread
replicas for you.** People assume it must — surely three replicas means three
nodes? No. The scheduler optimizes *fit* (resources, constraints), and several
forces actively encourage stacking: bin-packing scoring, image locality (the
node that already pulled your image scores higher), and in Moonlight's case a
resource crunch that made one node the only viable target. Spreading is a
*constraint you must declare*, not a default you receive.

## This cluster, right now

`website` runs 3 replicas — all pinned to one worker by a `nodeSelector`
somebody added months ago ("it was faster on that node"). Kill that node and
you've re-run Moonlight's morning.

```sh
kubectl -n kubelings get pods -l app=website -o wide   # NODE column: all identical
```

## Your task

Make one node's death survivable:

1. Remove the hard pin.
2. Declare spreading — anti-affinity or topology spread — so the 3 replicas land
   on **at least 2 distinct nodes** (this kind cluster has 2 workers + 1
   control-plane).
3. Fully Available afterward.

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch deploy website --type=json -p '[
  {"op":"remove","path":"/spec/template/spec/nodeSelector"}
]'
kubectl -n kubelings patch deploy website --type=strategic -p '
spec:
  template:
    spec:
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels: {app: website}
'
kubectl -n kubelings rollout restart deploy/website
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


## Root cause (theirs and yours)

Moonlight: correlated scheduling — nothing told the scheduler the replicas must
not share a failure domain, so under resource pressure they all fit best on one
node. Here: a forgotten `nodeSelector` did the same job more explicitly. Same
blast radius either way.

## The spreading toolbox

```yaml
# Modern default — topology spread:
topologySpreadConstraints:
  - maxSkew: 1                              # max imbalance between domains
    topologyKey: kubernetes.io/hostname     # domain = node (use .../zone for AZs)
    whenUnsatisfiable: ScheduleAnyway       # soft; DoNotSchedule = hard
    labelSelector: {matchLabels: {app: website}}
```

```yaml
# The older tool — pod anti-affinity:
podAntiAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:   # soft
    - weight: 100
      podAffinityTerm:
        labelSelector: {matchLabels: {app: website}}
        topologyKey: kubernetes.io/hostname
```

Judgment calls that matter:

- **Soft vs hard:** hard (`DoNotSchedule` / `required…`) guarantees spread but
  can deadlock scheduling — remember the access-modes lesson where a *required*
  rule created an unschedulable pod. Soft + monitoring usually wins.
- `maxSkew: 1` over N nodes ≈ even spread; anti-affinity is binary
  (together/not), spread constraints are proportional. Prefer spread for
  replicas > nodes.
- `…IgnoredDuringExecution` means existing pods aren't moved — spreading applies
  at (re)schedule. Hence the `rollout restart` in the fix.

## Prevention

- Standard template for any service with replicas ≥ 2: soft topology spread on
  hostname (and on zone, in multi-AZ clouds). Two stanzas, copy-paste.
- Audit for stacking: `kubectl get pods -o wide -l app=X` — NODE column all
  equal is a pre-incident, not a curiosity.
- Treat every `nodeSelector` in a Deployment as a code smell demanding a comment
  with an expiry date.

</details>
