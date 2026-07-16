> **Pattern drill.** Not one company's postmortem — a failure shape that appears
> in almost everyone's: logs, caches, tmp files, and image layers quietly eat
> node disks until the kubelet starts killing things. You'll hit the pod-level
> version live, and learn the node-level version it protects you from.

## The situation

`report-builder` warms a ~200Mi on-disk cache at startup. Watch it for a minute:

```sh
kubectl -n kubelings get pods -w
```

```
report-builder-...   1/1   Running   0     45s
report-builder-...   0/1   Evicted   0     52s
report-builder-...   1/1   Running   0     4s
```

**Evicted** is a status you haven't met. It isn't a crash — the process didn't
exit, there's no restart count, no CrashLoopBackOff. The **kubelet** killed the
pod, and the corpse stays behind with the reason written on it:

```sh
kubectl -n kubelings describe pod <an-evicted-one> | grep -B2 -A4 Evicted
```

```
Status:  Failed
Reason:  Evicted
Message: Pod ephemeral local storage usage exceeds the total limit
         of containers 64Mi.
```

Everything a container writes to its own filesystem — plus `emptyDir` volumes
and its logs — is **ephemeral storage**, drawn from the node's disk. It's the
fourth resource after CPU, memory, GPU… and the one nobody declares. This pod
has a 64Mi limit and a 200Mi appetite; the kubelet enforces the limit by
eviction. Then the ReplicaSet — which only counts *running* replicas — replaces
it, the warmup starts over, and you get an eviction treadmill that looks
almost healthy from a distance (`1/2 available`, forever).

Two things memory taught you that do **not** carry over:

- Memory overuse ⇒ OOMKill ⇒ *restart in place* (Module 2). Disk overuse ⇒
  **eviction** ⇒ pod deleted, rescheduled from scratch.
- OOMKilled pods vanish into restart counts; **Evicted pods linger as Failed
  objects** until a human (or a tuned GC threshold) deletes them.

## Your task

1. Budget the real footprint: raise the container's `ephemeral-storage` limit
   so a ~200Mi warmup fits with headroom (≥ 256Mi is honest).
2. Wait out one clean warmup — both replicas Running past the old kill line.
3. Bury the corpses: delete the Evicted (Failed) pods so `get pods` tells the
   truth again.
