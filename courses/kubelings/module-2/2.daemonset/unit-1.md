---
kind: unit
title: "Build a Node-Level Log Collector DaemonSet"
name: daemonset-unit
---


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

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings create -f - <<'EOF'
apiVersion: apps/v1
kind: DaemonSet
metadata: {name: node-logger}
spec:
  selector: {matchLabels: {app: node-logger}}
  template:
    metadata: {labels: {app: node-logger}}
    spec:
      containers:
        - name: agent
          image: ghcr.io/iximiuz/labs/busybox:latest
          command: ["sh","-c","while true; do echo collecting; sleep 3600; done"]
EOF
```

Control-plane nodes are schedulable in this playground, so no tolerations are
strictly required — but adding a control-plane toleration is good practice.

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
