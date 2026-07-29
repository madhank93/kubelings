> **Incident file (guided reading).** No lab: the failure lives in decisions made
> when the cluster was *created* — subnet masks, per-node pod caps, IPAM mode —
> and none of them can be changed by fixing a manifest afterwards. What's
> reproducible here is the arithmetic, and the arithmetic is the whole lesson.
>
> Sources:
> [loveholidays — When GKE ran out of IP addresses](https://deploy.live/blog/when-gke-ran-out-of-ip-addresses/) ·
> [Neon — postmortem: delayed compute starts (IP exhaustion + control-plane overload)](https://neon.com/blog/postmortem-delayed-start-compute-operations)

## What happened

loveholidays ran GKE in **VPC-native** mode, where every pod gets a real VPC
address from a subnet's *secondary range*. They had given pods a `/16` —
65,536 addresses. For a cluster of a few hundred nodes that is comically
oversized, right?

Then deploys started hanging. Pods sat **Pending**. The cluster autoscaler,
asked for more nodes, declined — and the error, when someone finally read it far
enough down the stack, said the subnet's IP space was exhausted. Roughly half of
newly created pods never reached Running.

## The arithmetic nobody does

GKE does not hand out pod IPs one at a time. It **pre-allocates a block per
node**, sized at *double* the node's max-pods setting, so that addresses aren't
reused the instant a pod dies (a reused IP is a connection delivered to the wrong
workload — see M4 `incident-conntrack` for what stale flow state does).

Default max-pods = 110 → **/24 reserved per node = 256 addresses**.

```
65,536 addresses ÷ 256 per node = 256 nodes.        ← the real ceiling
65,536 ÷ 110 pods = 595 nodes.                       ← the ceiling everyone assumes
```

The cluster hit a wall at less than half the size its owners had budgeted for,
and the wall was invisible until an autoscaler tried to cross it.

The same shape shows up everywhere the pod network is *routable*:

| Environment | Where the ceiling hides |
|---|---|
| GKE VPC-native | secondary range + 2× max-pods block per node |
| EKS + VPC CNI | ENI count × IPs-per-ENI **per instance type**; a `t3.small` caps at 11 pods |
| AKS Azure CNI | subnet must hold nodes × (max-pods + 1) |
| kubeadm + a routed CNI | `--pod-network-cidr` split into per-node `podCIDR`s (`/24` by default) — a `/16` cluster CIDR = 256 nodes, permanently |

## Why the symptom is confusing

Pods are Pending, so everyone opens the *scheduler* investigation: quotas,
taints, resource requests, affinity. But the scheduler is fine — it assigned the
pod to a node. The failure is one layer down, at **pod sandbox creation**, when
the CNI's IPAM has nothing to hand out:

```
Warning  FailedCreatePodSandBox  ... failed to allocate for range 0:
         no IP addresses available in range set: 10.244.3.1-10.244.3.254
```

That message is on the **pod's events**, not the scheduler's, and the
autoscaler's parallel failure ("cannot add node: IP space of subnetwork is
exhausted") arrives in a completely different log. Two symptoms, one cause, no
shared dashboard — which is why this incident class burns hours.

```sh
# What each node was actually given, and how full it is:
kubectl get nodes -o custom-columns=NAME:.metadata.name,PODCIDR:.spec.podCIDR,MAXPODS:.status.allocatable.pods
kubectl get pods -A --field-selector=status.phase=Pending -o wide
kubectl get events -A --field-selector reason=FailedCreatePodSandBox
```

## Concept checks

- Your cluster CIDR is `10.244.0.0/16` and each node gets a `/24`. What is the
  maximum node count, and what happens to the 111th pod on a node whose
  `allocatable.pods` is 110? (256 nodes; the pod stays Pending with
  `TooManyPods` — a *different* ceiling from the IP one, hit first or second
  depending on your node size.)
- Why does lowering **max-pods per node** free IP addresses even though you
  haven't deleted a single pod? (The reservation is per node and sized from
  max-pods, not from pods actually running. loveholidays got ~30% of their range
  back this way.)
- Which is cheaper to fix in place: an undersized *node* subnet or an undersized
  *pod* range? (Neither, really — but pod ranges can sometimes be *added* as
  extra secondary ranges, while the primary node subnet usually can't grow. This
  is why the sizing conversation belongs before `cluster create`.)

## What the industry took from it

- **Size the pod range from nodes × per-node block, not from pods.** Write the
  arithmetic down in the cluster's own docs; the person who scales the fleet in
  two years will not re-derive it.
- **Tune max-pods per node pool.** Stateful/cache/security pools rarely need 110
  pods; dropping the cap is the single highest-leverage IP saving, and it costs
  nothing.
- **Fewer, bigger nodes** stretch a fixed range further (one reservation per
  node) — the opposite of the usual "many small nodes" instinct, and a real
  trade against blast radius (M5 `incident-same-node`).
- **Alert on free IPs, not on pod count.** Cloud providers expose subnet
  utilisation metrics; a 70% threshold turns this outage into a ticket.
- **Non-routable pod networks dodge the whole problem** (overlay CNIs like
  Cilium/Calico in VXLAN mode, or GKE Autopilot's managed ranges) — at the cost
  of encapsulation and the debuggability you get from a flat network. Know which
  trade your cluster made.

*No check — study, then advance.*

<figure class="lesson-diagram">
<img src="/diagrams/incident-ip-exhaustion-ipam.svg" alt="incident-ip-exhaustion ipam diagram" loading="lazy">
</figure>

