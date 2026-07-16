> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern

## The situation

A teardown job is retiring an old environment. Everything deletes cleanly —
except:

```
NAME       STATUS        VOLUME     CAPACITY   ACCESS MODES   AGE
data-old   Terminating   pvc-4f2…   1Gi        RWO            94d
```

An hour now. CI is timing out. It's 2 a.m. somewhere and a voice in the incident
channel suggests: *"just strip the finalizers, I found a command on
StackOverflow."*

Do not. First, understand what's actually happening — because **nothing is
stuck**. Look:

```sh
kubectl -n kubelings get pvc data-old -o jsonpath='{.metadata.finalizers}'
```

```
["kubernetes.io/pvc-protection"]
```

Deletion in Kubernetes is a two-phase protocol: the object gets a
`deletionTimestamp` (that's your `Terminating`), then each **finalizer** — a
controller's registered veto — must be satisfied and removed before the object
actually goes. `pvc-protection`'s condition is simple: **no pod may still be
using this claim.** Some pod still is. The system is refusing to yank a disk out
from under a running process. That's not a bug. That's the seatbelt.

## Your task

Release the claim *properly*:

1. Find who still mounts `data-old` (`describe pvc` has a `Used By:` field).
2. Deal with the consumer.
3. Watch the PVC delete **itself** — you never touch the finalizer.

```sh
kubectl -n kubelings describe pvc data-old | grep -i "used by"
```

The check fails you if you strip finalizers. The seatbelt stays on.
