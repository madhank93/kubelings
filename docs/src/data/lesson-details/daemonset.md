## The situation

There's no per-node log collection. You need an agent that runs **one pod on
every node** in the cluster — and automatically on any node that joins later.
That workload shape is a **DaemonSet**, not a Deployment.

## Your task

In the `kubelings` namespace, create a DaemonSet named **`node-logger`**:

1. One pod per node (DaemonSets do this by design — no `replicas`).
2. Use image `ghcr.io/iximiuz/labs/busybox:latest`, command e.g.
   `sh -c 'while true; do echo collecting; sleep 3600; done'`.
3. All its pods must reach Ready.

```sh
kubectl -n kubelings get ds,pods -o wide
```
