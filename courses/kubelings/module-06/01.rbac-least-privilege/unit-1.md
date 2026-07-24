---
kind: unit
title: "cluster-admin for a bot: scope it down"
name: rbac-least-privilege-unit
---


## The situation

The security review finds this in ten seconds:

```sh
kubectl get clusterrolebinding ci-bot-admin -o wide
```

```
NAME           ROLE                        SUBJECTS
ci-bot-admin   ClusterRole/cluster-admin   kubelings/ci-bot
```

The CI bot's job: **read pod status in `kubelings`** to report build health. Its
actual powers: read every Secret in every namespace, delete kube-system, create
cluster-admin bindings for others, `kubectl exec` into anything. The archaeology
is always the same: something got 403 on a Friday, someone typed
`--clusterrole=cluster-admin`, the error went away, the grant stayed.

Why this is the highest-leverage fix in cluster security: **a ServiceAccount's
power is what its *token* can do, and tokens leak.** Every pod running as this
SA carries the token as a mounted file. Compromise any one of those pods — a
dependency CVE, a supply-chain artifact, an exposed debug port — and the
attacker holds cluster-admin. This exact chain, on a real company's clusters, is
the next lesson. Today you shrink the blast radius *before* the breach.

## Your task

Replace the god-grant with least privilege:

1. Delete `ci-bot-admin`.
2. Create a namespaced `Role` allowing `get`, `list`, `watch` on `pods` in
   `kubelings` — nothing else. **No secrets.**
3. Bind it to `ci-bot` with a `RoleBinding`.
4. Prove both directions with `kubectl auth can-i` — the check does.

```sh
SA=system:serviceaccount:kubelings:ci-bot
kubectl auth can-i --list --as=$SA -n kubelings | head   # before: everything
```

<details>
<summary>Hint</summary>

```sh
kubectl delete clusterrolebinding ci-bot-admin
kubectl -n kubelings create role pod-reader \
  --verb=get,list,watch --resource=pods
kubectl -n kubelings create rolebinding ci-bot-pod-reader \
  --role=pod-reader --serviceaccount=kubelings:ci-bot
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


## The four RBAC objects (keep them straight forever)

| | Namespaced | Cluster-wide |
|---|---|---|
| **what you may do** | Role | ClusterRole |
| **who may do it** | RoleBinding | ClusterRoleBinding |

Legit crossover: RoleBinding → ClusterRole grants the ClusterRole's verbs *only
within that namespace* — how you reuse standard roles (`view`, `edit`) without
cluster scope. A **ClusterRoleBinding**, by contrast, is everywhere-and-forever;
each one deserves an audit comment.

## Fix

```sh
kubectl delete clusterrolebinding ci-bot-admin
kubectl -n kubelings create role pod-reader --verb=get,list,watch --resource=pods
kubectl -n kubelings create rolebinding ci-bot-pod-reader \
  --role=pod-reader --serviceaccount=kubelings:ci-bot
```

Verify both directions — capability *and* incapability:

```sh
SA=system:serviceaccount:kubelings:ci-bot
kubectl auth can-i list pods -n kubelings --as=$SA        # yes ✓
kubectl auth can-i get secrets -n kubelings --as=$SA      # no ✓
kubectl auth can-i delete deploy -n kube-system --as=$SA  # no ✓
```

## Why "no secrets" got its own check

`get secrets` is quiet privilege escalation: other SAs' tokens live in Secrets
(legacy token Secrets, CI systems' stored creds). Read-secrets ≈
become-anyone-in-the-namespace. RBAC has no "all resources except secrets"
wildcard — which is by design: **enumerate what you grant.** Wildcards (`*`) in
Roles are the same Friday shortcut as cluster-admin, one layer down.

## Prevention

- `kubectl auth can-i --list --as=<sa>` in CI for your critical SAs — powers
  drift; pin them with a test.
- Audit standing ClusterRoleBindings quarterly:
  `kubectl get clusterrolebindings -o wide | grep -v system:` — every row needs
  an owner and a reason.
- Default-deny mindset: new bot = new Role with exactly its verbs. Expanding
  later is one PR; un-leaking a god token is an incident (next lesson).

</details>
