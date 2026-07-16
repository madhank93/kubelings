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
