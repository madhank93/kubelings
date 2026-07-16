## The situation

A vendor's Helm chart lands on your desk for approval. Buried in it:

```yaml
spec:
  hostNetwork: true
  containers:
    - securityContext:
        privileged: true
      volumeMounts:
        - {name: dockersock, mountPath: /var/run/docker.sock}
  volumes:
    - name: dockersock
      hostPath: {path: /var/run/docker.sock}
```

Read what that grants, line by line:

- `privileged: true` — the container runs with **all kernel capabilities**, sees
  host devices. The security boundary between container and node is essentially
  off.
- `hostNetwork: true` — shares the node's network namespace; sniff and bind host
  ports directly.
- **`hostPath` to `docker.sock`** — the crown jewel: write access to the
  container runtime's socket = *create any container on the node, as root,
  mounting the host filesystem.* This single mount is a full node takeover.

This pod isn't a workload; it's a **container escape wearing a YAML costume.** In
old Kubernetes, nothing stopped it — the API server admitted whatever you asked.

## Pod Security Standards

Kubernetes' built-in admission control for exactly this, three levels:

| Standard | Blocks |
|---|---|
| **privileged** | nothing (unrestricted) |
| **baseline** | the known-dangerous: privileged, hostNetwork/PID/IPC, most hostPath, adding capabilities |
| **restricted** | baseline **+** must run as non-root, drop ALL caps, seccomp, no privilege escalation |

You opt a namespace in with labels — `enforce` (reject), plus `warn`/`audit`
(allow but flag). No webhook to run; it's built into the API server.

## Your task

1. Enforce at least **baseline** on the `kubelings` namespace (a namespace
   label).
2. Rework or remove `vendor-agent` so no privileged/hostNetwork pod remains —
   baseline won't let it exist anyway.
3. The check proves enforcement by confirming a fresh privileged pod is
   **rejected**.

```sh
kubectl get ns kubelings --show-labels
kubectl -n kubelings get pod vendor-agent -o jsonpath='{.spec.containers[0].securityContext}'
```
