---
kind: unit
title: "Pod Security: the privileged pod that shouldn't exist"
name: pod-security-baseline-unit
---


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

<details>
<summary>Hint</summary>

```sh
# remove the dangerous pod
kubectl -n kubelings delete pod vendor-agent
# enforce the baseline (and get warnings on future violations)
kubectl label ns kubelings \
  pod-security.kubernetes.io/enforce=baseline \
  pod-security.kubernetes.io/warn=baseline --overwrite
```

Now try to create a privileged pod — the API server refuses it.

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


## Fix

```sh
kubectl -n kubelings delete pod vendor-agent
kubectl label ns kubelings \
  pod-security.kubernetes.io/enforce=baseline \
  pod-security.kubernetes.io/warn=baseline \
  pod-security.kubernetes.io/audit=baseline --overwrite
```

If the vendor agent has a legitimate job, it must justify each privilege and
almost always can run without them (a network agent rarely needs the Docker
socket). *"Vendor asked for privileged"* is the start of a security review, not
the end.

## Why enforce at the namespace, not review-by-eyeball

Kubelings' own runner treats lesson code as untrusted and confines it — same
philosophy. Human review misses one PR eventually; **enforced admission misses
zero.** The policy converts "please don't ship privileged pods" (a hope) into
"you cannot" (a guarantee). Prefer machine-enforced invariants over vigilance
for anything that matters.

## Rollout without breakage

Real clusters have existing violators, so stage it:

```sh
# 1. observe impact without blocking
kubectl label ns <ns> pod-security.kubernetes.io/warn=baseline \
  pod-security.kubernetes.io/audit=baseline --overwrite
# 2. fix what lights up
# 3. then enforce
kubectl label ns <ns> pod-security.kubernetes.io/enforce=baseline --overwrite
```

`warn`/`audit` first, `enforce` last — never flip enforce on a busy namespace
blind. Aim for `restricted` on anything running untrusted or internet-facing
code; `baseline` is the floor, not the goal.

## Prevention

- New namespace = PSS labels in the same manifest. Bake it into your namespace
  template / provisioning.
- Cluster-wide default via AdmissionConfiguration so an unlabeled namespace
  isn't silently `privileged`.
- Grep for the footguns in review:
  `grep -rE 'privileged: true|hostNetwork: true|/var/run/docker.sock|hostPath'`
  across incoming charts.

</details>
