---
kind: unit
title: "Drill — externalTrafficPolicy: Local blackholes half your nodes"
name: pattern-etp-blackhole-unit
---


> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters (cloud load balancers in front of `Local` Services,
> node drains, single-node placement), not a specific company's incident.

## The situation

`checkout` needs the **real client IP** — fraud scoring keys off it, and legal
wants it in the access log. So the Service was created with
`externalTrafficPolicy: Local`, which is the correct answer to that requirement:
traffic that lands on a node is delivered to a pod **on that same node**, with no
second hop and no SNAT, so the source address survives.

Then someone pinned all three replicas to one node.

```sh
kubectl -n kubelings get pods -o wide -l app=checkout   # NODE column: all the same
kubectl -n kubelings get svc checkout -o jsonpath='{.spec.externalTrafficPolicy}{"\n"}'
```

From one node, `:30090` answers. From every other node, it doesn't answer at
all — no reset you can attribute, no 502 from an ingress, no pod log. In a real
cluster the load balancer is what discovers this, and only if its health checks
are pointed at the right port. Otherwise it keeps sending a share of production
traffic to nodes that will silently drop it.

```sh
for ip in $(kubectl get nodes -l '!node-role.kubernetes.io/control-plane' \
              -o jsonpath='{range .items[*]}{.status.addresses[?(@.type=="InternalIP")].address}{" "}{end}'); do
  echo -n "$ip: "; curl -fsS -m 3 "http://$ip:30090/" >/dev/null 2>&1 && echo ok || echo BLACKHOLE
done
```

> Probe from a node that isn't the target: traffic a node generates *itself*
> reaches a NodePort under the cluster policy, so a node always appears healthy
> to itself even when it blackholes everyone else's packets.

<!-- d2:local -->
```text
     ┌────────────────────┐
     │traffic hits node B │
     │                    │
     └────────────────────┘
             │             
externalTrafficPolicy Local
             │             
             ▼             
    ┌────────────────────┐ 
    │node B hosts no pod │ 
    │                    │ 
    └────────────────────┘ 
             │             
     no local endpoint     
             │             
             ▼             
     ┌───────────────┐     
     │packet dropped │     
     │               │     
     └───────────────┘     
```
<!-- /d2:local -->

## Your task

Keep the client IP. Lose the blackhole:

1. `externalTrafficPolicy` stays **`Local`** — SNAT-ing it away with `Cluster` is
   the fix that breaks the requirement.
2. The Service stays a NodePort on **30090**.
3. **Every worker node** must answer on `:30090` — i.e. every node must host a
   ready `checkout` endpoint.
4. `checkout` keeps at least 2 available replicas.

<details>
<summary>Hint</summary>

A node forwards a `Local` Service only if it *hosts* an endpoint. So the question
is placement: drop the pin, then make the scheduler spread rather than pack.

```sh
kubectl -n kubelings patch deploy checkout --type=json \
  -p='[{"op":"remove","path":"/spec/template/spec/nodeSelector"}]'

kubectl -n kubelings patch deploy checkout --type=merge -p '{
  "spec": {
    "strategy": {"rollingUpdate": {"maxSurge": 0, "maxUnavailable": 1}},
    "template": {"spec": {"topologySpreadConstraints": [{
      "maxSkew": 1,
      "topologyKey": "kubernetes.io/hostname",
      "whenUnsatisfiable": "DoNotSchedule",
      "nodeTaintsPolicy": "Honor",
      "labelSelector": {"matchLabels": {"app": "checkout"}}
    }]}}
  }
}'
kubectl -n kubelings rollout status deploy/checkout
```

Two details that bite here, both worth the scar tissue:

- **`nodeTaintsPolicy: Honor`** — by default spreading counts *every* node as a
  domain, including the tainted control-plane your pods can never land on. A
  0-pod domain inflates the skew and the third replica goes Pending forever.
- **`maxSurge: 0`** — a hard spread constraint plus a surge pod is the M2.19
  rollout deadlock: the surge replica has nowhere legal to go, so the rollout
  parks. Replace before you add.

The other legitimate answer, and the one platform teams usually pick for
`Local` Services: run the workload as a **DaemonSet** so "one endpoint per node"
is structural instead of a scheduling wish.

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

## The pattern (why this recurs everywhere)

| | `Cluster` (default) | `Local` |
|---|---|---|
| Node with no local endpoint | forwards to a pod on another node | **drops** |
| Client IP | SNAT'd to the node — you see the node | **preserved** |
| Extra hop | yes (second-hop latency, cross-AZ cost) | no |
| Load balancing | even across all pods | per-node share, **skewed by placement** |

`Local` moves the correctness burden from kube-proxy to *you*: the guarantee it
gives — no rewrite of the source address — is paid for by the requirement that
every node in the rotation actually runs a pod.

The three ways clusters hit it:

1. **Placement** (this drill) — a nodeSelector, an affinity rule, or plain
   bin-packing leaves nodes endpoint-less.
2. **Drains** — `kubectl drain` evicts the last pod on a node while the external
   load balancer still has that node in rotation. Traffic to it dies until the LB
   health check catches up. (M8 `node-maintenance` is the drain side of this.)
3. **Scale-down** — an HPA scales 6 → 2 and four nodes quietly become blackholes.

## Why a real cloud LB usually saves you (and when it doesn't)

For `type: LoadBalancer` with `Local`, Kubernetes allocates a
**`healthCheckNodePort`** — a per-node endpoint that answers 200 only while the
node hosts a ready pod:

```sh
kubectl -n kubelings get svc checkout -o jsonpath='{.spec.healthCheckNodePort}{"\n"}'
```

The cloud LB is *supposed* to be pointed at that port, so endpoint-less nodes get
pulled from rotation within a health-check interval or two. It fails when: the LB
was hand-configured to health-check the app port (or `/`), the check interval is
long enough to matter during a rolling update, or — as here — the traffic isn't
coming through that LB at all (NodePort, an on-prem VIP, kube-vip, MetalLB in L2
mode).

## Fix

Give every node a local endpoint — DaemonSet, or a spread constraint plus enough
replicas — and keep the two facts wired together: `Local` + placement is one
decision, not two.

## Prevention / takeaway

- Grep for the combination and check its placement story:
  ```sh
  kubectl get svc -A -o json | jq -r '.items[]
    | select(.spec.externalTrafficPolicy=="Local")
    | "\(.metadata.namespace)/\(.metadata.name)"'
  ```
- Alert on the health-check port, per node — not just on aggregate LB 5xx.
- If you don't need the client IP, don't buy the constraint: `Cluster` is the
  boring, safe default. `Local` is a trade you should be able to name out loud.

</details>
