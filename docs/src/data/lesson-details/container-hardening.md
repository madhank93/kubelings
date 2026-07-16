## The situation

The `worker` deployment writes one heartbeat file every five seconds. Now look
at what it's *allowed* to do:

```sh
kubectl -n kubelings exec deploy/worker -- id
kubectl -n kubelings exec deploy/worker -- sh -c 'echo pwned > /bin/evil && ls -la /bin/evil'
```

```
uid=0(root) gid=0(root) groups=...
-rwxr-xr-x  1 root  root  6 ... /bin/evil
```

Root. A writable root filesystem (it just planted a binary in `/bin`). The
default Linux capability set. No syscall filter. **None of it is used by the
workload — all of it is available to whoever compromises the workload.** The
cryptominer from lesson 6.2 didn't exploit Kubernetes; it *used* exactly these
defaults after landing in a sloppy container.

Pod Security "restricted" (lesson 6.4) would refuse to admit this pod. This
lesson is the other side: writing the pod that *passes* — and understanding
what each field takes off the table.

## Your task

Harden `worker` until it runs with least privilege **and keeps working**:

| Field | Set to | Removes |
|---|---|---|
| `runAsNonRoot` + `runAsUser` | `true`, e.g. `10001` | root in the container = root-adjacent on the node |
| `readOnlyRootFilesystem` | `true` | planting binaries, editing configs |
| `capabilities.drop` | `["ALL"]` | raw sockets, chown, setuid… the root toolbox |
| `allowPrivilegeEscalation` | `false` | setuid/file-caps clawing privileges back |
| `seccompProfile.type` | `RuntimeDefault` | ~50 exotic syscalls (kernel attack surface) |

The catch that makes this a real skill: the app writes to `/work`. A read-only
root filesystem breaks that unless you **give it a writable volume** — an
`emptyDir` mounted exactly where writes happen, and nowhere else.
