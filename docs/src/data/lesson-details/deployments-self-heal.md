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
