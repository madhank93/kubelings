---
kind: unit
title: "seccomp on, AppArmor understood"
name: apparmor-seccomp-unit
---


## The situation

M6.6 (`container-hardening`) stripped capabilities, dropped root, made the
filesystem read-only. One door was left open: **syscalls**. The `audit-me`
pod has no `seccompProfile` — on most runtimes that means `Unconfined`: all
~350 Linux syscalls reachable, including the exotic ones (`keyctl`,
`userfaultfd`, `unshare`, `bpf`…) that kernel CVEs love and no web app has
ever needed. Container escapes almost always travel through a syscall the
workload never used.

**seccomp** is the kernel's syscall filter. Kubernetes exposes it in the
`securityContext`, and the zero-thought, high-value setting is
`RuntimeDefault` — containerd/CRI-O's curated profile (~60 syscalls denied)
that breaks essentially no normal workload.

```sh
kubectl -n kubelings get pod audit-me -o jsonpath='{.spec.securityContext}'
# {} — nothing
```

## Your task (checked)

Recreate `audit-me` with the runtime's default profile at pod level
(securityContext is immutable on a running pod — delete and recreate):

```yaml
spec:
  securityContext:
    seccompProfile:
      type: RuntimeDefault
```

Verify with the same jsonpath the checker uses:

```sh
kubectl -n kubelings get pod audit-me -o jsonpath='{.spec.securityContext.seccompProfile.type}'
# RuntimeDefault
```

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings delete pod audit-me
kubectl apply -n kubelings -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: audit-me
  labels: {app: audit-me}
spec:
  securityContext:
    seccompProfile: {type: RuntimeDefault}
  containers:
    - name: audit-me
      image: busybox:1.36
      command: ["sh", "-c", "while true; do sleep 30; done"]
EOF
```

Container-level `securityContext.seccompProfile` also passes the check —
it overrides pod-level for that container.

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

---

## Reading: AppArmor — the host-side twin

> **Runbook reading.** AppArmor profiles live in the *node's* kernel and are
> loaded with root on the host — outside this course's kubectl sandbox. This
> is the exact sequence for when you have node access (any Linux kind node
> via `docker exec`, or an iximiuz playground VM). CKS expects it.

Where seccomp filters *syscalls*, **AppArmor** (Debian/Ubuntu family; SELinux
on RHEL) confines *what those syscalls may touch*: file paths, capabilities,
network. The two stack — RuntimeDefault seccomp + a tight AppArmor profile
is defense in depth on the same pod.

**1 · Write a profile** (deny writes everywhere except `/tmp`):

```
# /etc/apparmor.d/k8s-deny-write
#include <tunables/global>
profile k8s-deny-write flags=(attach_disconnected) {
  #include <abstractions/base>
  file,
  deny /** w,
  owner /tmp/** w,
}
```

**2 · Load it into the node's kernel** (every node that may run the pod —
this is why real fleets bake profiles into the node image or ship them with
a DaemonSet):

```sh
apparmor_parser -r /etc/apparmor.d/k8s-deny-write
aa-status | grep k8s-deny-write        # loaded?
```

**3 · Reference it from the pod.** Since v1.30 it's first-class in
`securityContext` (the old `container.apparmor.security.beta.kubernetes.io/…`
annotation still appears in older clusters and exam questions):

```yaml
spec:
  securityContext:
    appArmorProfile:
      type: Localhost
      localhostProfile: k8s-deny-write
```

**4 · Prove it from inside:**

```sh
kubectl exec <pod> -- cat /proc/1/attr/current
# k8s-deny-write (enforce)
kubectl exec <pod> -- touch /etc/x
# touch: /etc/x: Permission denied      ← the profile, not file perms
```

**Failure mode worth knowing:** reference a profile that isn't loaded on the
scheduled node and the pod is stuck `Blocked`/`CreateContainerError` — a
scheduling-dependent failure that looks random until you check which nodes
carry the profile.

## Takeaway

- `RuntimeDefault` seccomp is the cheapest real attack-surface cut in
  Kubernetes — it belongs in every pod template, enforced by admission
  policy (M6.12's Kyverno can require exactly this field).
- AppArmor completes it on the host side: seccomp says *which* syscalls,
  AppArmor says *on what*.
- Both are per-node kernel state referenced by pod spec — the referencing is
  kubectl; the loading is node ops.
- Custom seccomp profiles (`type: Localhost` + a JSON allowlist) exist for
  the paranoid tier; measure first with `RuntimeDefault` + audit logs before
  hand-rolling one.
