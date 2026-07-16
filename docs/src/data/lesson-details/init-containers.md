## The situation

```
NAME                       READY   STATUS     RESTARTS   AGE
reports-7f6b8d9c4d-x2kwm   0/1     Init:0/1   0          22m
```

`Init:0/1` — the pod hasn't even *started* being your application yet. Init
containers run **before** app containers, strictly in order, each to completion.
Until the last one exits 0, the app containers don't exist. That's the feature:
a programmable gate for "don't start until X is true."

This gate is waiting for a config file to appear in a mounted volume… and it has
been "waiting for config volume..." for 22 minutes. The ConfigMap it wants
*exists*. So why is the volume empty?

Look closely at the two names in the pod spec.

## Your task

Get `reports` Running and Available:

1. Read the init container's logs — init containers have logs too
   (`kubectl logs <pod> -c <init-container-name>`).
2. Compare the volume's ConfigMap reference against what actually exists.
3. Fix the reference.

```sh
kubectl -n kubelings get pods -l app=reports
kubectl -n kubelings logs -l app=reports -c wait-for-config --tail=5
kubectl -n kubelings get configmaps
kubectl -n kubelings get deploy reports -o jsonpath='{.spec.template.spec.volumes}'
```
