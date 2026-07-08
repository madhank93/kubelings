---
kind: unit
title: "Service types: open a door to the outside"
name: nodeport-vs-clusterip-unit
---


## The situation

Demo day. The `demo` Service works beautifully — from inside the cluster.
`ClusterIP` is exactly that: a **virtual IP that exists only in cluster
routing**. The client on the guest Wi-Fi types it into a browser and gets
nothing, because outside the cluster that IP is fiction.

The Service type ladder, each rung adding exposure:

| Type | Adds | Reachable from |
|---|---|---|
| **ClusterIP** | stable VIP + DNS | pods only (default, correct for most things) |
| **NodePort** | same port opened on **every node** | anyone who can reach any node IP |
| **LoadBalancer** | cloud LB pointing at the NodePorts | the internet (cloud-managed) |

`LoadBalancer` needs a cloud controller — on bare kind it stays `<pending>`
forever. Today's tool is **NodePort**: kube-proxy programs every node to forward
a high port (default range 30000–32767) into the Service.

## Your task

1. Make Service `demo` type `NodePort` with `nodePort: 30080` (pinned, so the
   check — and your client — knows where to knock).
2. Confirm it answers on a node's InternalIP:30080.

```sh
kubectl -n kubelings get svc demo
kubectl get nodes -o wide    # InternalIP column
```

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch svc demo --type=merge -p '
spec:
  type: NodePort
  ports:
    - port: 80
      targetPort: 80
      nodePort: 30080
'
NIP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
wget -qO- http://$NIP:30080/
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


## What NodePort actually does

kube-proxy adds a rule **on every node** — including nodes running zero demo
pods: *"packets to me:30080 → demo endpoints."* Hit any node, reach the service;
the mesh routes across nodes internally. That's why it works and why it's
wasteful: every node donates a port from a small shared range (30000–32767,
~2700 doors for the whole cluster).

## The ladder in practice

- **ClusterIP** — default. If nothing outside the cluster needs it, stop here.
  Every extra rung is attack surface (ask JW Player — Module 6).
- **NodePort** — demos, on-prem edges, and the substrate LoadBalancers build on.
  Couples clients to node IPs, which change.
- **LoadBalancer** — production external traffic, one cloud LB per Service
  (cost!). On kind: pending forever, no cloud controller.
- Beyond the ladder: **Ingress/Gateway** — one LB, many routes, L7. Different
  lesson.

## Fix

The patch from the hint. Check the wiring end to end:

```sh
kubectl -n kubelings get svc demo
# PORT(S): 80:30080/TCP  ← service port : node port
```

## Prevention / habits

- Pin `nodePort` only when something external hardcodes it (like this check);
  otherwise let Kubernetes pick and avoid range collisions.
- Audit exposure regularly: `kubectl get svc -A | grep -v ClusterIP` — every row
  is a door to the outside. Every door should have a reason you can say out loud.

</details>
