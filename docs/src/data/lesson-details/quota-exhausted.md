## The situation

You scaled `workers` to 5. Then:

```
NAME      READY   UP-TO-DATE   AVAILABLE
workers   2/5     2            2
```

Stuck at 2/5. No pods are crashing — the missing three **were never created.**
`kubectl get pods` shows two happy pods and no failures, which is exactly what
makes this one baffling: *the failure isn't on a pod, because the pods don't
exist.* Follow the ownership chain up:

```sh
kubectl -n kubelings describe rs -l app=workers | grep -A6 Events
```

```
Warning  FailedCreate  ... Error creating: pods "workers-..." is forbidden:
  exceeded quota: kubelings-quota, requested: requests.memory=64Mi,
  used: requests.memory=128Mi, limited: requests.memory=160Mi
```

A **ResourceQuota** caps the *total* resources a namespace may request. This one
allows 160Mi of memory requests; two 64Mi pods use 128Mi; a third would hit 192
> 160, so admission **rejects the pod creation**. The ReplicaSet controller keeps
trying and keeps getting 403 — the error surfaces on the *controller*, never on a
pod, because no pod ever makes it past admission.

## Your task

Get `workers` to 5/5. Two honest paths — pick one:

- raise the quota to fit the workload, or
- lower the workload's footprint to fit the quota (smaller requests, or fewer
  replicas if 5 was aspirational).

```sh
kubectl -n kubelings get resourcequota kubelings-quota -o yaml
kubectl -n kubelings describe resourcequota kubelings-quota   # Used vs Hard
```
