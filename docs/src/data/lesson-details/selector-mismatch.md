## The situation

This one has everyone stumped, because *everything looks fine*.

The `api` pods: Running, 2/2, zero restarts. The `api` Service: exists, correct
port. DNS: `api.kubelings.svc` resolves instantly. And yet every single request:

```
wget: can't connect to remote host: Connection timed out
```

A teammate already "checked everything" and is now blaming the CNI plugin. It is
not the CNI plugin. It's almost never the CNI plugin.

There's one command nobody ran:

```sh
kubectl -n kubelings get endpoints api
```

```
NAME   ENDPOINTS   AGE
api    <none>      3m
```

`<none>`. The Service is a perfectly healthy signpost pointing at **zero pods**.
DNS resolves the Service's virtual IP fine — but behind that IP, the routing
table is empty, so connections go nowhere and time out.

## Your task

Get traffic flowing to `api`:

1. Compare what the Service *selects* with what the pods *are labeled*.
2. Fix the mismatch (fix the Service **or** the pod labels — your call).
3. Endpoints must show both pod IPs, and an in-cluster request must succeed.

```sh
kubectl -n kubelings get svc api -o jsonpath='{.spec.selector}'
kubectl -n kubelings get pods -l app=api --show-labels
kubectl -n kubelings get endpoints api
```
