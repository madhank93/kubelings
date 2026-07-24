---
kind: unit
title: "Hardening: take away everything the workload doesn't use"
name: container-hardening-unit
---


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

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch deploy worker --type=strategic -p '
spec:
  template:
    spec:
      containers:
        - name: worker
          securityContext:
            runAsNonRoot: true
            runAsUser: 10001
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            capabilities: {drop: ["ALL"]}
            seccompProfile: {type: RuntimeDefault}
          volumeMounts:
            - {name: work, mountPath: /work}
      volumes:
        - {name: work, emptyDir: {}}
'
```

If the pod crashloops after hardening, `kubectl logs` tells you which
permission it actually needed — that's the discovery loop working as intended.

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


## Why each field, in attacker terms

Think of a compromise as a shopping trip; each field empties a shelf:

- **Non-root**: most container escapes and node-level mischief start from
  uid 0. A random high uid also defeats "matching uid on the host" tricks.
- **Read-only rootfs**: malware wants persistence — dropped binaries, edited
  startup scripts. Read-only means it lives only in memory and dies with the
  pod (which restarts constantly, per Module 1).
- **Drop ALL capabilities**: capabilities are root's powers unbundled;
  workloads that serve HTTP need approximately none. (The classic exception:
  binding ports <1024 wants `NET_BIND_SERVICE` — add back *only* that.)
- **allowPrivilegeEscalation: false**: sets the kernel's `no_new_privs` bit —
  without it, a setuid binary in the image can silently undo your uid work.
- **seccomp RuntimeDefault**: filters the syscall surface — the kernel bugs
  reachable from a container drop sharply. Cost in practice: ~zero for normal
  apps, which is why "restricted" PSS simply requires it.

Together these are most of what the **Pod Security "restricted"** profile
checks. The pyramid so far: RBAC (6.1) limits the *API*, tokens (6.5) limit
*identity*, PSS (6.4) *enforces* this lesson at admission, and these fields
limit the *kernel*.

## Beyond pod fields (the CKS horizon)

Fields in the pod spec can only restrict what the *default runtime* offers.
Two stronger moves live at the node/runtime layer, out of kubectl's reach on
this cluster but worth knowing cold:

- **AppArmor/SELinux**: per-program mandatory access control (file paths, not
  just syscalls). AppArmor profiles attach via
  `securityContext.appArmorProfile` — but the profile itself must exist on
  the node first.
- **Sandboxed runtimes** (gVisor, Kata): a `RuntimeClass` object routes a pod
  to a runtime that puts a userspace kernel or a microVM between container
  and host — the strongest isolation short of a separate node. Untrusted or
  multi-tenant workloads live there.

## Prevention

- Bake this securityContext into your base manifests/kustomize base (last
  lesson) — hardening as the default, exceptions as reviewed diffs.
- Enforce with PSS `restricted` on the namespace (6.4) so unhardened pods
  can't even land.
- The discovery loop for legacy apps: harden → run → read the failure → grant
  back the *one* thing needed (a writable mount, one capability) → repeat.
  It converges fast, and the result is documentation of what the app truly
  requires.

</details>
