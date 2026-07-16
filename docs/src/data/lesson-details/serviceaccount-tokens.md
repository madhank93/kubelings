## The situation

The security review after the cryptominer incident (lesson 6.2) flagged one
line item nobody could explain: *"why does the marketing site have an API
credential?"*

It does. Exec into any `web` pod and look:

```sh
kubectl -n kubelings exec deploy/web -- ls /var/run/secrets/kubernetes.io/serviceaccount/
```

```
ca.crt  namespace  token
```

That `token` is a signed credential for the Kubernetes API, projected into the
pod automatically because **every pod runs as a ServiceAccount** — if you don't
name one, it's `default`, and by default the token is **automounted** into every
container. nginx will never call the API; the credential is pure attack surface.
In the cryptominer incident this is exactly what got monetized: compromise any
container, read the token, *become* its ServiceAccount:

```sh
# What the token in a web pod is worth right now:
kubectl auth can-i --list --as=system:serviceaccount:kubelings:default -n kubelings
```

Meanwhile the one workload that *legitimately* needs the API — `audit-agent`,
which lists pods — is broken in the opposite direction. Someone wrote it a
least-privilege Role and RoleBinding (lesson 6.1 style), but:

```sh
kubectl -n kubelings get rolebinding audit-agent-reads-pods -o yaml | grep -A3 subjects
kubectl -n kubelings get sa
```

The binding's subject is a ServiceAccount named `audit-agent` — **which doesn't
exist**. And the deployment never sets `serviceAccountName`, so it runs as
`default` anyway. The grant points at a ghost; the workload carries the wrong
badge.

## Your task

Three moves — give the agent its identity, wire it in, and take the default
credential away from everyone else:

1. Create the `audit-agent` ServiceAccount the RoleBinding already expects.
2. Set `serviceAccountName: audit-agent` in the `audit-agent` pod template
   (this triggers a rollout — the token is projected at pod creation).
3. Set `automountServiceAccountToken: false` on the **default** ServiceAccount,
   so pods that never ask for an identity never carry one.

Prove it the same way an attacker would probe it:

```sh
kubectl auth can-i list pods --as=system:serviceaccount:kubelings:audit-agent -n kubelings
```
