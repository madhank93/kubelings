---
kind: unit
title: "CreateContainerConfigError: the key that isn't there"
name: configmap-wiring-unit
---


## The situation

A new status in your zoo:

```
NAME                       READY   STATUS                       RESTARTS   AGE
pricing-6b9d8c7f4d-tk3mx   0/1     CreateContainerConfigError   0          4m
```

Not `CrashLoopBackOff` (the app never ran — no logs). Not `ImagePullBackOff`
(the image pulled fine). `CreateContainerConfigError` means the kubelet pulled
the image, then tried to *assemble the container's configuration* — env vars,
mounted keys — and couldn't. The pod is stuck **before** process start, and the
answer lives in events:

```sh
kubectl -n kubelings describe pod -l app=pricing | tail -6
```

```
Error: couldn't find key taxRate in ConfigMap kubelings/pricing-config
```

The ConfigMap exists. It has `tax_rate`. The Deployment asks for `taxRate`.
snake_case vs camelCase — a one-key typo, and the kubelet retries forever.

## Your task

Get `pricing` Running with `TAX_RATE=0.19` actually inside the container:

```sh
kubectl -n kubelings get configmap pricing-config -o yaml
kubectl -n kubelings get deploy pricing -o jsonpath='{.spec.template.spec.containers[0].env}'
```

Fix the mismatch from whichever side you think is right — but know that other
services may read this ConfigMap too (changing the consumer is safer than
changing the shared key).

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch deploy pricing --type=json -p '[
  {"op":"replace","path":"/spec/template/spec/containers/0/env/1/valueFrom/configMapKeyRef/key","value":"tax_rate"}
]'
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

`configMapKeyRef` asked for key `taxRate`; the ConfigMap holds `tax_rate`. The
kubelet cannot construct the env, so the container never starts. (Same failure
shape for Secrets: `couldn't find key … in Secret …`.)

## Fix

```sh
kubectl -n kubelings patch deploy pricing --type=json -p '[
  {"op":"replace","path":"/spec/template/spec/containers/0/env/1/valueFrom/configMapKeyRef/key","value":"tax_rate"}
]'
kubectl -n kubelings rollout status deploy/pricing
```

## env vs mount — the difference that bites later

| | `env` / `envFrom` | volume mount |
|---|---|---|
| read | **copied at container start** | file read at access time |
| ConfigMap edited | pod keeps old values until restarted | file updates in place (~1 min sync) |
| typo'd key | `CreateContainerConfigError` | mounts empty/missing file (silent!) |

Both have failure modes: env fails **loudly at start**, mounts fail **quietly at
runtime**. Pick env for boot-time constants, mounts for config you intend to
reload — and remember editing a ConfigMap **never** restarts consumers; pair
config changes with `kubectl rollout restart` or a checksum annotation.

## Prevention

- One naming convention per ConfigMap (snake_case *or* camelCase), enforced in
  review.
- `kubectl describe pod` first for any `Create…Error` — the event names the
  exact missing key and object.

</details>
