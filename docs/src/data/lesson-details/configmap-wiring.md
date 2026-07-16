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
