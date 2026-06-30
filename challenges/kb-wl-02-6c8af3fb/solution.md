# Solution — node-logger DaemonSet

## Approach

A per-node agent is a **DaemonSet** — it schedules exactly one pod per eligible
node and tracks node add/remove automatically.

## Create it

```sh
kubectl -n kubelings apply -f - <<'EOF'
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-logger
spec:
  selector:
    matchLabels: {app: node-logger}
  template:
    metadata:
      labels: {app: node-logger}
    spec:
      tolerations:
        - key: node-role.kubernetes.io/control-plane
          operator: Exists
          effect: NoSchedule
      containers:
        - name: agent
          image: ghcr.io/iximiuz/labs/busybox:latest
          command: ["sh","-c","while true; do echo collecting; sleep 3600; done"]
EOF
```

## Verify

```sh
kubectl -n kubelings get ds node-logger
kubectl -n kubelings get pods -l app=node-logger -o wide
```

`DESIRED` should equal `READY` and match the node count.
