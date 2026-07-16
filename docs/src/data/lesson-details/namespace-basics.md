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
