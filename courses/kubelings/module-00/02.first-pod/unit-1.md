---
kind: unit
title: "Your first pod: what a cluster actually runs"
name: first-pod-unit
---

## The situation

A Kubernetes **cluster** is a group of machines, called **nodes**, plus a
control plane that decides what runs where. You never log into a node to start
a program. You tell the control plane what you want with `kubectl`, and it
finds a node with room and starts it there.

The smallest thing Kubernetes runs is a **pod**: one or more containers that
share a network address and are always scheduled together. Most pods hold one
container.

Look around first. Every command below is read-only:

```sh
kubectl get nodes            # the machines in the cluster
kubectl get namespaces       # folders that group objects by name
kubectl get pods -A          # every pod, in every namespace
```

The pods in `kube-system` are Kubernetes itself (DNS, networking, the control
plane). Your work goes in the `kubelings` namespace.

## Your task

1. Start a pod named `hello` in the `kubelings` namespace, running the image
   `nginx:1.27-alpine`.
2. Watch it until its `STATUS` is `Running`.
3. Find out which node it landed on.

```sh
kubectl -n kubelings run hello --image=nginx:1.27-alpine
kubectl -n kubelings get pods --watch        # Ctrl-C to stop watching
```

<details>
<summary>Hint</summary>

`-n kubelings` (or `--namespace kubelings`) picks the namespace. Leave it out
and the pod lands in `default`, where the check will not find it.

`kubectl get pods -o wide` adds a `NODE` column.

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

```sh
kubectl -n kubelings run hello --image=nginx:1.27-alpine
kubectl -n kubelings get pod hello -o wide
```

## What just happened

1. `kubectl` sent a Pod object to the API server.
2. The **scheduler** picked a node and wrote its name into the pod.
3. The **kubelet** on that node pulled the image and started the container.

The status column shows those steps: `Pending` (no node yet, or image still
pulling), then `ContainerCreating`, then `Running`.

## Pods are not pets

Once the check is green, delete the pod. It is gone for good, because nothing
recreates it:

```sh
kubectl -n kubelings delete pod hello
```

That is why real workloads are almost never bare pods. A later lesson in this
module hands the job of keeping pods alive to a Deployment.

</details>
