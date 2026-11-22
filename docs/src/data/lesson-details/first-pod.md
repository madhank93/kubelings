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
