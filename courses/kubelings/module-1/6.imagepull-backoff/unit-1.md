---
kind: unit
title: "ImagePullBackOff: the tag that never existed"
name: imagepull-backoff-unit
---


## The situation

Friday, 16:58. A one-line hotfix ships. The deploy pipeline goes green — it only
checks that `kubectl apply` succeeded — and everyone goes home.

Monday:

```
NAME                        READY   STATUS             RESTARTS   AGE
frontend-7c9f6d4b8b-2mhx4   0/1     ImagePullBackOff   0          2d
frontend-7c9f6d4b8b-p8wvz   0/1     ImagePullBackOff   0          2d
```

`ImagePullBackOff` is the kubelet saying: *"I asked the registry for this image
and it said no — I'll retry with backoff."* The pod never even started; there is
no process, no logs. Which is the tell: **crash loops have logs, pull failures
have events.**

```sh
kubectl -n kubelings describe pod -l app=frontend | grep -A4 -i events
```

```
Failed to pull image "nginx:1.27.9999-alpine":
  ... nginx:1.27.9999-alpine: not found
```

The registry is up. The `nginx` repo exists. The *tag* `1.27.9999-alpine` was a
fat-fingered version that never existed — and `kubectl apply` will happily ship
a reference to nothing.

## Your task

Get `frontend` Available (2/2):

1. Confirm from events *why* the pull fails (don't guess — read it).
2. Point the Deployment at a tag that exists (`1.27-alpine` is fine).
3. Watch the rollout replace the stuck pods.

```sh
kubectl -n kubelings get pods -l app=frontend
kubectl -n kubelings describe pod -l app=frontend | tail -12
```

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings set image deploy/frontend frontend=nginx:1.27-alpine
kubectl -n kubelings rollout status deploy/frontend
```

`kubectl rollout undo deploy/frontend` also works when a previous ReplicaSet had
a good image — that's the real-world muscle memory for a bad deploy.

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


## Root cause

The image *reference* was wrong: `nginx:1.27.9999-alpine` names a tag that never
existed. The API server can't validate registry contents at apply time, so the
mistake only surfaces when the kubelet actually pulls — as an event, not a log.

## Triage pattern

| Symptom | Where the answer lives |
|---|---|
| `CrashLoopBackOff` | `kubectl logs --previous` (the process ran and spoke) |
| `ImagePullBackOff` / `ErrImagePull` | `kubectl describe pod` events (it never ran) |
| `CreateContainerConfigError` | events (missing ConfigMap/Secret reference) |

## Fix

```sh
kubectl -n kubelings set image deploy/frontend frontend=nginx:1.27-alpine
kubectl -n kubelings rollout status deploy/frontend
```

or roll back to the last working ReplicaSet:

```sh
kubectl -n kubelings rollout undo deploy/frontend
```

## Prevention

- Pipelines must gate on `kubectl rollout status`, not on `apply` exit code —
  apply succeeding means the *object* was accepted, not that the app runs.
- Pin images by digest (`nginx@sha256:…`) for immutability; tags can be typoed,
  moved, or deleted.
- Rollouts keep the old ReplicaSet around precisely so `rollout undo` is instant.

</details>
