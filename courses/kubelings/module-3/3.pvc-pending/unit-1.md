---
kind: unit
title: "PVC Pending: a claim nobody answers"
name: pvc-pending-unit
---


## The situation

The analytics team's new database has a pod and a disk request. The pod:

```
analytics-7d9c8b6f5d-2wkxm   0/1   Pending   0   31m
```

Pending — not even scheduled. And the reason chains one level down:

```sh
kubectl -n kubelings get pvc
```

```
NAME             STATUS    VOLUME   CAPACITY   ACCESS MODES   STORAGECLASS   AGE
analytics-data   Pending                                      fast-ssd       31m
```

A **PersistentVolumeClaim** is exactly what it sounds like: a *claim ticket* for
storage. Someone must answer it with a **PersistentVolume**. In modern clusters
that someone is a **StorageClass** — a named recipe pointing at a provisioner
that creates volumes on demand.

This claim names the class `fast-ssd`. Copy-pasted from the cloud cluster's
manifests, where `fast-ssd` maps to premium disks. Here?

```sh
kubectl get storageclass
```

No `fast-ssd`. The claim is addressed to a department that doesn't exist in this
building. Nothing errors — dynamic provisioning simply never begins, and PVC and
pod wait forever, politely.

## Your task

Get the PVC **Bound** and `analytics` Running with a writable `/data`:

1. See what StorageClasses this cluster actually offers.
2. Point the claim at one that exists (or at the default).
3. Note: PVC spec is mostly immutable — you'll need to recreate the claim, not
   patch it.

```sh
kubectl get storageclass
kubectl -n kubelings describe pvc analytics-data | tail -5
```

<details>
<summary>Hint</summary>

kind ships the `standard` class (local-path provisioner), marked default:

```sh
kubectl -n kubelings delete pvc analytics-data
kubectl apply -n kubelings -f - <<'EOF'
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: analytics-data
spec:
  accessModes: [ReadWriteOnce]
  storageClassName: standard
  resources:
    requests: {storage: 1Gi}
EOF
```

(Or omit `storageClassName` entirely → the default class answers.) The Pending
pod picks the new claim up automatically. `standard` uses
`WaitForFirstConsumer`, so the PVC binds the moment the pod schedules.

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


## The triangle

```
PVC (I need 1Gi, RWO, class X)
 └─ StorageClass X (recipe: provisioner, params, binding mode)
     └─ provisioner creates PV → PVC binds → pod mounts
```

Break any edge and the claim sits Pending: nonexistent class (this lesson),
provisioner down, or an unsatisfiable request (size, access mode).

## Fix

Recreate the claim against a class that exists (see hint). Two behaviors worth
internalizing:

- **PVC immutability:** you can't repoint an existing claim's class — delete and
  recreate. That's why the fix feels heavier than editing a Deployment.
- **`WaitForFirstConsumer`:** the PVC stays Pending *by design* until a pod uses
  it, so the volume lands in the right topology (same node/zone as the pod).
  Don't panic-debug a Pending PVC that simply has no consumer yet — check
  `kubectl describe pvc` for `waiting for first consumer`.

## Diagnosis one-liners

```sh
kubectl get sc                                  # what can answer claims here
kubectl -n <ns> describe pvc <name> | tail -5   # WHY it's Pending (events)
kubectl get pv                                  # what's been provisioned
```

## Prevention

- Never hardcode cloud StorageClass names in portable manifests — omit the field
  and let each cluster's default answer, or template it per environment.
- Alert on PVCs Pending > 5 min *that have a consumer* — that's always a
  provisioning failure, never normal.

</details>
