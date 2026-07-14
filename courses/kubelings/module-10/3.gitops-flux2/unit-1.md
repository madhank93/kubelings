---
kind: unit
title: "Flux: the Kustomization that can't find its source"
name: gitops-flux2-unit
---


## The situation

Same GitOps promise as Argo CD (10.1), different architecture. **Flux** is
not one controller but a *toolkit* of small ones, each owning one CRD, each
consuming the previous one's output:

```
GitRepository ──(artifact: a tarball of the repo at the resolved ref)──▶ Kustomization ──▶ cluster
   source-controller                                          kustomize-controller
```

Where Argo CD is an integrated app with a UI, Flux is composable plumbing —
closer in spirit to how Kubernetes itself is built (M7: small controllers,
each reconciling one kind). podinfo should be running. It isn't:

```sh
kubectl -n kubelings get gitrepositories,kustomizations
# NAME         READY   STATUS
# podinfo      False   failed to checkout and determine revision: unable to
#                      clone: couldn't find remote ref "refs/heads/production"
# NAME         READY   STATUS
# podinfo      False   Source artifact not found, retrying in 30s
```

Read the chain the way Flux runs it, **source first**:

- `GitRepository` pins `branch: production` — the repo has no such branch
  (`git ls-remote https://github.com/stefanprodan/podinfo | head` — it's
  `master` plus tags). No resolvable ref → no artifact.
- The `Kustomization` isn't broken at all — it's *starving*: its
  `sourceRef` points at a source that never produced an artifact. Fixing
  the downstream error message directly would be treating the symptom.

Two objects, one root cause, and the conditions tell you which is which —
the same "fix the cause, not the stuck consumer" shape as M3's
pattern-namespace-terminating.

## Your task

1. Fix the ref:

   ```sh
   kubectl -n kubelings patch gitrepository podinfo --type=merge \
     -p '{"spec":{"ref":{"branch":"master"}}}'
   ```

2. Watch the chain go Ready in order — source, then kustomization (each
   reconciles on its `interval`, ~1m):

   ```sh
   kubectl -n kubelings get gitrepositories,kustomizations -w
   kubectl -n kubelings get deploy podinfo
   ```

<details>
<summary>Hint</summary>

Impatient? Annotate to reconcile *now* (this is what the `flux reconcile`
CLI does under the hood):

```sh
kubectl -n kubelings annotate gitrepository podinfo \
  reconcile.fluxcd.io/requestedAt=$(date +%s) --overwrite
```

A real deployment would pin `ref: {tag: 6.7.1}` or a semver range rather
than a branch — same rule as Argo's targetRevision: pin to something
immutable, but something that *exists*.

</details>

::simple-task
---
:tasks: tasks
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>


## Fix

```sh
kubectl -n kubelings patch gitrepository podinfo --type=merge \
  -p '{"spec":{"ref":{"branch":"master"}}}'
```

## Flux vs Argo CD, honestly

| | Flux | Argo CD |
|---|---|---|
| Shape | toolkit of single-purpose controllers | integrated app (+UI, +RBAC, +SSO) |
| Unit of intent | GitRepository + Kustomization / HelmRelease | Application |
| Multi-repo/fleet | Kustomizations referencing Kustomizations | app-of-apps / ApplicationSets |
| Debugging | per-CRD `.status.conditions`, chained | one `.status` ladder per Application |
| Install here | pinned `install.yaml` (bootstrap wants repo write access) | pinned `install.yaml` |

Both converge git → cluster; the differences are operational taste. What
transfers between them: sources must resolve, pins must exist, and the
first controller in the chain with a False condition owns the root cause.

## Prevention / takeaway

- **Debug in dependency order.** Flux's chained CRDs make "source first"
  explicit — a Kustomization error that mentions artifacts is almost never
  the Kustomization's fault.
- `prune: true` makes git deletions real deletions — the Flux flavor of
  drift correction (Argo's selfHeal). Enable it deliberately; it will
  delete what you remove from the repo.
- Branch pins rot (renames, deletions); tag/semver pins don't. CI-check
  every ref with `git ls-remote` — one line, catches this whole lesson.
- The reconcile-now annotation beats waiting out `interval` during
  incidents; put it in the runbook.

</details>
