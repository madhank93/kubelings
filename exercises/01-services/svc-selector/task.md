# Service selector mismatch

A `web` Deployment is running fine — two nginx pods, both Ready. But the `web`
Service in front of them sends traffic **nowhere**: it has no endpoints, so
anything calling `web` gets connection refused.

A Service finds its pods by **label selector**. If the selector doesn't match the
pods' labels, the Service selects nothing.

## Your task

Make the `web` Service route to the `web` pods, in the `kubelings` namespace.

## Inspect

```sh
kubectl -n kubelings get deploy,pods --show-labels
kubectl -n kubelings get svc web -o yaml
kubectl -n kubelings get endpointslices -l kubernetes.io/service-name=web
```

Notice the pods are labelled `app=web`, but the Service selects something else.

## Done when

The `web` Service has at least one **ready endpoint** (i.e. its selector matches
the running pods). The check polls for this.

> Tip: edit the Service's `spec.selector` and re-apply, then run verify.
> `kubectl -n kubelings edit svc web` works too.
