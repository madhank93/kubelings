---
title: "Pattern: PVC stuck Terminating"
description: "[PATTERN] Synthetic composite — a PersistentVolumeClaim that will not delete because a finalizer is waiting on a pod nobody remembers."
---

> **[PATTERN] scenario** — a synthetic composite of a failure mode reported
> across many production clusters. **No specific company**; details are
> representative, not cited. (Real, cited incidents are marked `[REAL]` in the
> [Incident Library](/reference/incident-library/).)

## Situation

A teardown job is cleaning up a retired staging environment. Everything deletes
— except one PersistentVolumeClaim, which has been sitting like this for an hour:

```
NAME       STATUS        VOLUME     CAPACITY   AGE
data-old   Terminating   pv-4f21b   50Gi       94d
```

Deletes "hang", CI times out, someone suggests force-deleting things at 2 a.m.

## Root cause

The PVC carries the `kubernetes.io/pvc-protection` **finalizer**. Kubernetes
will not fully delete a claim while any pod still mounts it — the deletion is
accepted (hence `Terminating`) but completion is blocked until the finalizer is
removed by its controller.

Somewhere, a forgotten pod — a paused debug pod, a stuck Job, a CronJob run that
never got cleaned up — still references the claim. The PVC isn't broken; it's
*protecting you from yanking a disk out from under a running process.*

## Diagnosis

```sh
# who still mounts it?
kubectl -n staging describe pvc data-old | grep -A5 "Used By"
# or search every pod's volumes:
kubectl -n staging get pods -o json \
  | jq -r '.items[] | select(.spec.volumes[]?.persistentVolumeClaim.claimName=="data-old") | .metadata.name'
```

## Fix

Delete (or finish) the consumer pod first; the finalizer clears and the PVC
deletes itself seconds later:

```sh
kubectl -n staging delete pod <consumer-pod>
kubectl -n staging get pvc data-old   # gone
```

**Anti-pattern:** `kubectl patch pvc data-old -p '{"metadata":{"finalizers":null}}'`.
Stripping finalizers "works" and is how people corrupt volumes — the pod keeps
writing to a disk whose claim no longer exists, and the PV's reclaim policy may
fire while data is in flight. Treat finalizer-stripping as a last resort for
truly orphaned objects, never as the standard unblock.

## Prevention

- Teardown order matters: workloads → claims → volumes.
- Set `ttlSecondsAfterFinished` on Jobs so finished pods don't linger as ghost
  consumers.
- Alert on objects in `Terminating` for longer than a threshold — it always
  means a finalizer is waiting on something you can name.

## What it teaches

| Concept | Kubelings module |
|---|---|
| PVC/PV lifecycle & protection finalizers | M3 Config & Storage |
| Finalizers & deletion mechanics | M7 Internals |
