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
