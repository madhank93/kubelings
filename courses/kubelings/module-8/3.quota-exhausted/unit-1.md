---
kind: unit
title: "Deploy blocked: the quota nobody mentioned"
name: quota-exhausted-unit
---


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

<details>
<summary>Hint</summary>

See how full it is, then give it room:

```sh
kubectl -n kubelings describe resourcequota kubelings-quota
kubectl -n kubelings patch resourcequota kubelings-quota --type=merge \
  -p '{"spec":{"hard":{"requests.memory":"512Mi","limits.memory":"1Gi"}}}'
```

The ReplicaSet retries continuously — the remaining pods appear within seconds.

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


## Two namespace-scoped limiters (don't confuse them)

| | **ResourceQuota** | **LimitRange** |
|---|---|---|
| scope | **total** across the namespace | **per-pod/container** defaults & bounds |
| stops | namespace exceeding an aggregate cap | individual pods too big/small, or with no requests |
| this lesson | ✅ hit the aggregate cap | — |

A subtle rule you just met: once a namespace has a quota on `requests.memory`,
**every** pod must *declare* memory requests or it's rejected — the quota forces
explicitness. Pair a ResourceQuota with a LimitRange (default requests) so
existing bare pods don't suddenly fail admission.

## Where this class of failure hides

The symptom is on the **controller**, not the workload. The general skill:
**walk the ownership chain toward the error.**

```
Deployment (2/5, no obvious error)
  └─ ReplicaSet  ← the FailedCreate event lives HERE
      └─ Pods    ← don't exist, so nothing to see
```

`kubectl describe deploy` → `describe rs` → events. Anything "won't create the
pods" (quota, PodSecurity rejection from Module 6, a validating webhook) shows up
on the ReplicaSet/Job, never on a phantom pod.

## Prevention

- Quota-managed namespaces need a companion LimitRange (default requests/limits)
  so workloads without explicit resources don't silently fail admission.
- Alert on `FailedCreate` events and on quota utilization > ~80% — you want to
  raise the ceiling *before* a scale-up during an incident hits it.
- On-call reflex: "deployment stuck below desired, pods not crashing" →
  `describe rs`, read the event. It's quota, PodSecurity, or a webhook ~every
  time.

</details>
