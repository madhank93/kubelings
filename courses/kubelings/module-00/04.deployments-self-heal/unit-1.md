---
kind: unit
title: "Deployments: let Kubernetes keep your pods alive"
name: deployments-self-heal-unit
---

## The situation

A pod is not restarted elsewhere if its node dies, and nothing recreates it if
someone deletes it. For anything that should stay up, you create a
**Deployment** instead and let it manage the pods.

A Deployment holds a pod **template** and a **replica count**. It creates a
**ReplicaSet**, and the ReplicaSet's only job is to keep exactly that many
matching pods in existence. Delete one and a replacement appears within
seconds. This is the pattern that runs through all of Kubernetes: you declare
the desired state, and a **controller** keeps working until the cluster matches.

```text
Deployment web  ──owns──▶  ReplicaSet web-7c9d…  ──owns──▶  pod web-7c9d…-abcde
  replicas: 3                 keeps 3 alive                  pod web-7c9d…-fghij
  template: nginx                                            pod web-7c9d…-klmno
```

## Your task

1. Create a Deployment named `web` in `kubelings`, image `nginx:1.27-alpine`.
2. Delete its pod and watch a new one replace it.
3. Scale it to **3** replicas and wait until all three are ready.

```sh
kubectl -n kubelings create deployment web --image=nginx:1.27-alpine
kubectl -n kubelings get deploy,rs,pods
```

<details>
<summary>Hint</summary>

Pods made by a Deployment get generated names, so copy one from `get pods`:

```sh
kubectl -n kubelings delete pod <name-from-get-pods>
kubectl -n kubelings get pods --watch
```

Scaling changes the replica count on the Deployment, not on a pod:

```sh
kubectl -n kubelings scale deployment web --replicas=3
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

```sh
kubectl -n kubelings create deployment web --image=nginx:1.27-alpine
kubectl -n kubelings delete pod -l app=web        # watch it come back
kubectl -n kubelings scale deployment web --replicas=3
kubectl -n kubelings rollout status deployment web
```

## The same thing as a manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: kubelings
spec:
  replicas: 3
  selector:
    matchLabels: {app: web}      # which pods this Deployment owns
  template:                      # what each pod looks like
    metadata:
      labels: {app: web}         # must match the selector
    spec:
      containers:
        - name: nginx
          image: nginx:1.27-alpine
```

The `selector` and the template's `labels` must match. That pairing of labels
and selectors is how a Deployment, and later a Service, finds its pods. Module 1
has a lesson where they don't match.

## Scaling the declarative way

`kubectl scale` is quick, but the file is the record. Change `replicas:` in the
manifest and `kubectl apply -f` it, and the next person to read the file sees
the real count.

</details>
