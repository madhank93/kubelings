---
kind: unit
title: "CrashLoopBackOff: read the logs before you guess"
name: crashloop-triage-unit
---


## The situation

It's your first on-call shift. The team just shipped a new `orders` service to the
`kubelings` namespace and Slack is already unhappy: *"orders is down, can you look?"*

You check — and it's the classic sight:

```
NAME                      READY   STATUS             RESTARTS      AGE
orders-6f7d9c5b8d-4x2lp   0/1     CrashLoopBackOff   4 (12s ago)   2m
orders-6f7d9c5b8d-tk9wz   0/1     CrashLoopBackOff   4 (18s ago)   2m
```

The image builds fine. It ran on the developer's laptop. Kubernetes even says the
pod *starts* — it just dies within a second, every time, and the backoff delay
between attempts keeps doubling.

`CrashLoopBackOff` is not an error. It's Kubernetes telling you: *"the container
keeps exiting, and I'm pausing between retries."* The actual error is somewhere
else — and there is exactly one reliable place to find it.

## Your task

Make `orders` run steadily (2/2 Available, no CrashLoopBackOff):

1. Look at *why* the container exits — its exit code and last state.
2. Read what the process itself said on the way down.
3. Fix the Deployment accordingly. Don't touch the image — it's fine.

```sh
kubectl -n kubelings get pods -l app=orders
kubectl -n kubelings describe pod -l app=orders | grep -A5 -i 'last state'
kubectl -n kubelings logs -l app=orders --previous --tail=20
```

<details>
<summary>Hint</summary>

`kubectl logs --previous` shows the output of the *last crashed* container — the
one that actually failed — not the current retry:

```
FATAL: required environment variable APP_MODE is not set
```

The app wants an environment variable. Give it one:

```sh
kubectl -n kubelings set env deploy/orders APP_MODE=production
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


## Root cause

The container's entrypoint checks for `APP_MODE` and exits `1` when it's missing.
Nobody set it in the Deployment. The kubelet dutifully restarts the container,
each exit-within-seconds increments the restart counter, and the restart policy's
exponential backoff produces `CrashLoopBackOff`.

Nothing was wrong with the image, the node, or Kubernetes. The process told you
exactly what it needed — on stderr, right before dying.

## The triage loop (memorize this)

```sh
kubectl -n kubelings get pods -l app=orders            # what state?
kubectl -n kubelings describe pod -l app=orders        # exit code, events
kubectl -n kubelings logs -l app=orders --previous     # what did it SAY?
```

`--previous` is the key: the *current* container is often mid-restart with no
output yet. The crashed one had the answer.

## Fix

```sh
kubectl -n kubelings set env deploy/orders APP_MODE=production
kubectl -n kubelings rollout status deploy/orders
```

or `kubectl -n kubelings edit deploy orders` and add under the container:

```yaml
env:
  - name: APP_MODE
    value: production
```

## Verify

```sh
kubectl -n kubelings get pods -l app=orders   # 1/1 Running, restarts stop growing
kubectl -n kubelings logs -l app=orders       # "orders service started in mode: production"
```

## Prevention

- Fail fast **with a clear message** (this app did — that's what saved you).
- Declare required config in the manifest, not tribal knowledge: env vars from a
  ConfigMap reviewed alongside the Deployment.
- A readiness/startup probe won't stop a crashloop, but it stops a half-booted
  pod from receiving traffic while you debug.

</details>
