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
