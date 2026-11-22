---
kind: unit
title: "Manifests: write down what you want, then apply it"
name: pod-manifests-unit
---

## The situation

`kubectl run` is fine for a quick test, but nobody can review it, put it in git,
or run it again next week. Real clusters are described in **manifests**: YAML
files that say what should exist. `kubectl apply -f file.yaml` sends the file to
the API server, and Kubernetes makes the cluster match it.

Every manifest has the same four top-level fields:

```yaml
apiVersion: v1          # which version of the API defines this kind
kind: Pod               # what sort of object
metadata:               # who it is: name, namespace, labels
  name: web
spec:                   # what you want it to be
  containers: [...]
```

**Labels** are key/value tags in `metadata.labels`. They do nothing on their
own, but Services, Deployments and `kubectl get -l` all find pods by label, so
almost every pod has them.

YAML is indentation-sensitive. Use spaces, never tabs, and keep items of the
same list at the same depth.

## Your task

Save this as `web.yaml`, fill in the three `TODO` lines, and apply it:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: web
  namespace: kubelings
  labels:
    app: web
    # TODO: add a second label, tier, with the value frontend
spec:
  containers:
    - name: web
      image: nginx:1.27-alpine
      ports:
        - containerPort: 0      # TODO: nginx listens on port 80
      resources:
        limits:
          memory: 64Mi          # TODO: raise the limit to 128Mi
```

```sh
kubectl apply -f web.yaml
kubectl -n kubelings get pods --show-labels
```

<details>
<summary>Hint</summary>

The new label goes directly under `app: web`, indented the same amount.

Made a mistake? Edit the file and run `kubectl apply -f web.yaml` again.
A few fields of a running pod cannot change (the port is one of them). If
`apply` says the change is forbidden, delete the pod and apply again:

```sh
kubectl -n kubelings delete pod web
kubectl apply -f web.yaml
```

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

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: web
  namespace: kubelings
  labels:
    app: web
    tier: frontend
spec:
  containers:
    - name: web
      image: nginx:1.27-alpine
      ports:
        - containerPort: 80
      resources:
        limits:
          memory: 128Mi
```

```sh
kubectl apply -f web.yaml
kubectl -n kubelings get pods -l tier=frontend
```

## Why `apply`

`kubectl apply` is **declarative**: the file is the record of what you want.
Run it twice and nothing changes the second time. Change the file and apply
again, and Kubernetes changes only what differs. `kubectl create` fails if the
object already exists, and `kubectl run` leaves no file behind.

## Reading any object back as YAML

```sh
kubectl -n kubelings get pod web -o yaml
```

The output is longer than what you wrote. Kubernetes fills in defaults and adds
a `status:` section describing what is actually happening. Your manifest is the
`spec`, the desired state. `status` is the observed state.

</details>
