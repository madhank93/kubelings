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
