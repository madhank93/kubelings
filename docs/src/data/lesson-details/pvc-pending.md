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
