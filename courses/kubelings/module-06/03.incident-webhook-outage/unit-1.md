---
kind: unit
title: "Incident replay — the webhook that froze the cluster (Jetstack)"
name: incident-webhook-outage-unit
---


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

<details>
<summary>Hint</summary>

Fastest safe fix — fail open:

```sh
kubectl patch validatingwebhookconfiguration policy-guard --type=json -p '[
  {"op":"replace","path":"/webhooks/0/failurePolicy","value":"Ignore"}
]'
```

Or, if the webhook is simply broken and unowned, remove it:

```sh
kubectl delete validatingwebhookconfiguration policy-guard
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


## Root cause chain (Jetstack's, generalized)

1. Webhook backend becomes unreachable (node recycle / crash / scale-to-zero).
2. `failurePolicy: Fail` → API server rejects all matching writes.
3. Scope is broad (`*` resources) and includes namespaces needed for recovery.
4. Recovery requires writes → writes are blocked → **deadlock.**

Every link is individually reasonable. The outage is their product — the same
"multiplication, not one bug" shape as the Zalando DNS incident.

## Building webhooks that can't do this

- **`failurePolicy: Ignore`** unless you can *prove* the backend is more
  available than the thing it guards. Most policy checks should fail open;
  a bypassed check during a rare outage beats a global write freeze.
- **Scope tightly:** real `rules` (specific apiGroups/resources), and a
  `namespaceSelector` / `objectSelector` that **excludes kube-system and the
  webhook's own namespace** — so it can never block its own recovery. `["*"]`
  on everything is the config equivalent of `chmod -R 777`.
- **`timeoutSeconds` low** (1–5s): a slow webhook shouldn't add latency to
  every write in the cluster.
- **HA + PDB for the backend**, and keep it off nodes it might itself gate.

## Two-way-door habit

Before shipping any webhook, answer: *"if this backend is down for 10 minutes,
what still works?"* If the answer is "nothing", you've built the Jetstack
outage. Also know the manual escape hatch for 3 a.m.:
`kubectl delete validatingwebhookconfiguration <name>` (or
`mutatingwebhookconfiguration`) — cluster-scoped, no admission on the webhook
configs themselves, so it works even when everything else is frozen.

## Why Security, not Internals

The mechanism is pure control-plane internals (you'll trace the admission chain
in Module 7). But it's here because webhooks are how you *enforce security
policy* — and the #1 way security tooling causes the outage it was meant to
prevent. Powerful guard, footgun safety off by default.

</details>
