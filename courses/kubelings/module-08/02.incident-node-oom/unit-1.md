---
kind: unit
title: "Incident replay — the node that OOMed itself (Blue Matador)"
name: incident-node-oom-unit
---


## The real incident

**Blue Matador** published a clean postmortem of a failure mode that hits
clusters running workloads with no memory limits: a node ran out of memory, and
the **Linux kernel OOM-killer** — which operates below Kubernetes, at the node
level — started killing processes to survive. It doesn't respect your intentions;
it kills by its own heuristic, and it can take out kubelet, container runtime,
or system daemons, cascading a single greedy pod into a **whole-node outage**
(`SystemOOM`).

Source: [Post Mortem: Kubernetes Node OOM — Blue Matador](https://www.bluematador.com/blog/post-mortem-kubernetes-node-oom)

Two very different OOMs — the distinction is the lesson:

| | **Pod OOM** | **Node OOM (`SystemOOM`)** |
|---|---|---|
| trigger | container exceeds *its* `limits.memory` | *node* runs out of physical memory |
| killer | kubelet/cgroup, surgically | kernel OOM-killer, by its own score |
| blast radius | that one container restarts | random processes; node destabilized |
| your control | you set the limit | you *avoid* it by setting limits everywhere |

You met pod OOM in Module 2 (`oomkill`) — bounded, almost tidy. This is its
feral cousin: with **no limit**, a container is free to grow until the *node*
starves, and then the kernel — not Kubernetes — decides who dies.

## This cluster, right now

`log-shipper` has memory **requests but no limits**. Nothing caps its growth at
the pod level. On a busy day it's one leak away from re-running Blue Matador's
incident on whichever node it lands on — taking its innocent neighbors with it.

```sh
kubectl -n kubelings get pods -l app=log-shipper \
  -o custom-columns=NAME:.metadata.name,LIM:.spec.containers[0].resources.limits.memory
```

## Your task

Cap the blast radius before the kernel has to:

1. Give every `log-shipper` container a memory **limit**.
2. Keep it Available.

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings set resources deploy/log-shipper \
  --requests=memory=32Mi --limits=memory=128Mi
```

Now a runaway container hits its own limit and gets a *pod* OOM (restart) long
before the node is endangered.

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


## Why "requests but no limits" is the dangerous combo

- **Requests** are for *scheduling* — the scheduler reserves that much and packs
  nodes by summing requests. They do **not** cap usage.
- **Limits** are for *enforcement* — the cgroup caps real usage; exceed it and
  the container (not the node) is OOM-killed.

Requests without limits means the scheduler packs the node believing everyone
uses ~their request, while any pod is actually free to consume all node memory.
Over-commit + a leak = node OOM. Memory is **incompressible**: unlike CPU (which
just throttles — Module 5), you cannot "throttle" memory. The only outcomes are
"have it" or "kill something."

## Fix + the node's own defenses

Set limits (done above). Know the layers underneath you:

- **kubelet eviction** tries to act *before* the kernel: on memory pressure it
  taints the node `MemoryPressure`, stops new pods, and **evicts** pods —
  BestEffort first, then Burstable over their request, Guaranteed last (the QoS
  order from Module 2, now doing real work). Limits everywhere make this orderly
  instead of a kernel massacre.
- **kube-reserved / system-reserved** carve out memory for the kubelet and OS so
  workloads can't starve the daemons that keep the node alive.

## Prevention

- **Memory limits on every container. Always.** (CPU limits are debatable —
  Module 5; memory limits are not.)
- Enforce with a **LimitRange** per namespace injecting default limits, so "no
  limit" is impossible, plus **ResourceQuota** to bound total namespace usage.
- Alert on `node_memory_MemAvailable` low and on `SystemOOM`/eviction events —
  and on any pod running without limits (`kubectl get pods -A -o json | jq …`).
- The recognition: **random unrelated pods dying + node flapping NotReady** = a
  node-OOM cascade, not N separate bugs. One unbounded pod, many victims.

</details>
