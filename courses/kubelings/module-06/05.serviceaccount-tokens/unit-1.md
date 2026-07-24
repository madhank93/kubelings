---
kind: unit
title: "The token in every pod (and who's using yours)"
name: serviceaccount-tokens-unit
---


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

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings create serviceaccount audit-agent
kubectl -n kubelings patch deploy audit-agent --type=merge \
  -p '{"spec":{"template":{"spec":{"serviceAccountName":"audit-agent"}}}}'
kubectl -n kubelings patch serviceaccount default --type=merge \
  -p '{"automountServiceAccountToken": false}'
```

Existing `web` pods keep their already-mounted token until they're recreated —
automount is decided at admission, not retroactively. Roll them if you want the
change to bite now: `kubectl -n kubelings rollout restart deploy/web`.

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


## What that token actually is

Since v1.22 the automounted token is a **projected, bound token**, not a
long-lived Secret: the kubelet requests it via the TokenRequest API, it
**expires** (~1h, auto-refreshed in place), and it's **audience- and pod-bound**
— stolen and replayed elsewhere, it's rejected once the pod is gone. That's a
real improvement over the old forever-valid Secret tokens, but bound-and-expiring
is damage *limitation*, not damage *prevention*: for its lifetime, the token is
the ServiceAccount. The fix is still "don't hand it to pods that don't need it."

The identity string you grant against is always:

```
system:serviceaccount:<namespace>:<name>
```

That's the thread tying this module together: RBAC (6.1) decides what an
identity may do; **this lesson decides which pods hold which identity**.

## The decision table

| Workload | ServiceAccount | Token mounted? |
|---|---|---|
| never calls the API (most apps) | `default` | **no** — automount off |
| calls the API | its **own** SA, one per workload | yes, with a narrow Role |
| needs cluster-wide reads (rare) | own SA + ClusterRole | yes — and reviewed |

One SA per API-calling workload is the audit unit: when a token shows up in an
audit log doing something weird, the SA name tells you *which deployment* is
compromised. Everything sharing `default` tells you nothing.

## Root cause / fix / prevention

- **Root cause:** identity was never designed — the workload inherited
  `default`, the grant referenced a name nobody created. RBAC subjects are not
  validated against existing ServiceAccounts; a binding to a ghost is silently
  useless (same failure shape as the selector mismatch in Module 1: two halves
  that only work if the names agree, and nothing checks).
- **Fix:** create the SA, set `serviceAccountName`, flip `automountServiceAccountToken: false`
  on `default`.
- **Prevention:**
  - `automountServiceAccountToken: false` on the default SA of **every**
    namespace at creation time (bake it into your namespace template).
  - A pod that needs the API opts in explicitly: own SA, own Role, automount on
    that SA only.
  - Audit occasionally: `kubectl auth can-i --list --as=system:serviceaccount:<ns>:default`
    should be nearly empty everywhere.
  - Pod Security (6.4) constrains what a pod can do to the *node*; this lesson
    constrains what it can do to the *API*. You need both — the cryptominer
    (6.2) walked through the second door.

</details>
