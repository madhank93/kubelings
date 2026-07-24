---
kind: unit
title: "Namespaces: walls, names, and crossing them"
name: namespace-basics-unit
---


## The situation

The platform team's request seems simple: *"We need a `staging` environment. Same
app config as production. And staging services must still be able to call the
production config API while we migrate."*

Your cluster currently has everything in `kubelings`: a `config-api` Deployment,
its Service, and an `app-config` ConfigMap. There is no `staging`.

Namespaces are Kubernetes' unit of *naming and policy* isolation. Inside one, a
name like `app-config` is unique; across them, it's free to repeat. RBAC, quotas
and limits attach per-namespace. But — and this trips everyone — **namespaces do
not isolate the network** by default. Any pod can reach any Service in any
namespace, as long as it uses the right name.

That name is the FQDN: `<service>.<namespace>.svc.cluster.local`. Within your own
namespace you say `config-api`; from another, you say
`config-api.kubelings.svc.cluster.local` (or the shorter `config-api.kubelings`).

## Your task

1. Create the `staging` namespace.
2. Copy the `app-config` ConfigMap from `kubelings` into `staging`.
3. Confirm a pod in `staging` can reach `config-api.kubelings.svc.cluster.local`
   — the check does exactly this.

```sh
kubectl get ns
kubectl -n kubelings get configmap app-config -o yaml
```

<details>
<summary>Hint</summary>

Export → strip namespace-specific metadata → re-apply:

```sh
kubectl create namespace staging
kubectl -n kubelings get configmap app-config -o yaml \
  | grep -v -E 'namespace:|resourceVersion:|uid:|creationTimestamp:' \
  | kubectl -n staging apply -f -
```

Test cross-namespace DNS yourself:

```sh
kubectl -n staging run tmp --rm -it --restart=Never --image=busybox:1.36 \
  -- wget -qO- http://config-api.kubelings.svc.cluster.local/
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


## What namespaces give you

| Isolated per-namespace | NOT isolated |
|---|---|
| object names | network reachability (needs NetworkPolicy) |
| RBAC scope | nodes / kernel |
| ResourceQuota, LimitRange | cluster-scoped objects (nodes, PVs, CRDs) |

## Fix

```sh
kubectl create namespace staging
kubectl -n kubelings get configmap app-config -o yaml \
  | grep -v -E 'namespace:|resourceVersion:|uid:|creationTimestamp:' \
  | kubectl -n staging apply -f -
```

## The DNS ladder

From a pod in `staging`, all of these resolve `config-api` in `kubelings`:

```
config-api.kubelings                      # svc + namespace
config-api.kubelings.svc                  # + svc marker
config-api.kubelings.svc.cluster.local    # fully qualified
```

Bare `config-api` would look in *staging* first — and fail there. Which is
exactly how "works in one namespace, 404 in another" bugs are born.

## Prevention / habits

- Treat namespace as part of a Service's identity: write FQDNs in config that
  crosses namespaces.
- Namespaces are not a security boundary for traffic — that's Module 4's
  NetworkPolicy territory.

</details>
