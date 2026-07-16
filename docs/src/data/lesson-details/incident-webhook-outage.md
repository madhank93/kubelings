## The real incident

**Jetstack** (the cert-manager company — people who know Kubernetes cold) took a
**GKE cluster outage from a single admission webhook.** A node auto-repair
recycled the nodes running the webhook's backend pods. The webhook was
configured `failurePolicy: Fail` with broad scope — so with its backend
briefly unreachable, the API server did what that policy commands: **rejected
every CREATE/UPDATE it was asked to admit.** Including the writes needed to
reschedule the webhook's own pods. The cluster had locked itself out of its own
recovery.

Source: [How a simple admission webhook lead to a cluster outage — Jetstack](https://blog.jetstack.io/blog/gke-webhook-outage)

Admission webhooks sit **in the synchronous write path** of the API server:
every matching write pauses while the API server calls your webhook. That gives
them veto power over the entire control plane. `failurePolicy` decides what
happens when the call *fails*:

- `Fail` — no answer = **reject** the write. "Secure", and a live grenade if the
  backend can ever be unavailable.
- `Ignore` — no answer = **allow** the write. Fails open; the policy isn't
  enforced during the outage, but the cluster keeps breathing.

## This cluster, right now

`policy-guard` is a ValidatingWebhookConfiguration with `failurePolicy: Fail`,
`resources: ["*"]`, pointed at a Service with **no pods behind it**. The
`kubelings` namespace is labeled into its scope. Result: every write to
`kubelings` is rejected — try one and read the error.

```sh
kubectl -n kubelings create configmap test --from-literal=a=b
# Error ... failed calling webhook "policy-guard...": ... no endpoints available
kubectl get validatingwebhookconfiguration policy-guard -o yaml | grep -A3 failurePolicy
```

## Your task

Restore writes to `kubelings` **and** leave the webhook config in a safe state
(the check rejects leaving it `Fail` with a dead backend):

- Flip `failurePolicy` to `Ignore` (fail-open) so an unreachable webhook can't
  block the cluster, **or** delete/rescope the broken configuration.
- Confirm you can create objects in `kubelings` again.

```sh
kubectl get validatingwebhookconfiguration policy-guard -o yaml | grep -A6 -i 'failurePolicy\|namespaceSelector'
```
