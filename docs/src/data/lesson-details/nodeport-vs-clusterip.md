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
