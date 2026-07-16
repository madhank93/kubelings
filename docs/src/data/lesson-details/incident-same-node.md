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
