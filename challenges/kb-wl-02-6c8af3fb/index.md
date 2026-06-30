---
kind: challenge

title: "Build a Node-Level Log Collector DaemonSet"
description: |
  Ops needs a log shipper running on every node — exactly one pod per node, now
  and on any node added later. Build a DaemonSet that satisfies this and confirm
  it lands a Ready pod on each node.

categories:
- kubernetes

tagz:
- cka
- workloads
- daemonset

difficulty: easy

createdAt: 2026-06-30

playground:
  name: k8s-omni

tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 180
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      kubectl wait --for=condition=Ready nodes --all --timeout=120s || true

  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      kubectl -n "$NS" get daemonset node-logger >/dev/null 2>&1 || {
        echo "not yet: no DaemonSet 'node-logger' in $NS"; exit 1; }
      desired=$(kubectl -n "$NS" get ds node-logger -o jsonpath='{.status.desiredNumberScheduled}')
      ready=$(kubectl -n "$NS" get ds node-logger -o jsonpath='{.status.numberReady}')
      if [ "${desired:-0}" -lt 1 ]; then
        echo "not yet: DaemonSet desires 0 pods — is it scheduled on nodes?"; exit 1
      fi
      if [ "${ready:-0}" -ne "${desired:-0}" ]; then
        echo "not yet: $ready/$desired DaemonSet pods Ready"; exit 1
      fi
      echo "PASS — node-logger is Ready on all $desired node(s)."
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
