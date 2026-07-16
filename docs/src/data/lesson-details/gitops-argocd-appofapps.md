## The situation

One Application was a pet (10.1). Real platforms run *fleets* — and the
fleet dashboard is a single kubectl:

```sh
kubectl -n argocd get applications
# NAME            SYNC STATUS   HEALTH STATUS
# shop-backend    Synced        Healthy
# shop-frontend   Unknown       Healthy      ← there
# shop-reports    Synced        Healthy
```

Three Applications deploy the shop, ordered by **sync waves** — check the
annotations:

```sh
kubectl -n argocd get applications -o custom-columns='NAME:.metadata.name,WAVE:.metadata.annotations.argocd\.argoproj\.io/sync-wave,REV:.spec.source.targetRevision'
# shop-backend    0   HEAD
# shop-frontend   1   stable
# shop-reports    2   HEAD
```

Waves order work *within* a sync: wave 0 must be Synced+Healthy before
wave 1 starts — dependency ordering (databases before apps before
dashboards) expressed as data, not pipeline scripts.

`shop-frontend` pins `targetRevision: stable`. The lesson-1 reflex:

```sh
kubectl -n argocd get application shop-frontend -o jsonpath='{.status.conditions}'
# ComparisonError … ambiguous argument 'stable': unknown revision or path…
```

The repo has no `stable` branch — someone imported a convention from a
different repo. The child can't render, so it can never be Synced.

## The pattern this drill is a slice of

In production these three children wouldn't be applied by hand — a
**parent Application** points at a git directory *containing Application
manifests*, so Argo CD manages Argo CD:

```yaml
# the parent — "app of apps"
spec:
  source:
    repoURL: https://github.com/your-org/platform
    path: apps/           # a directory of Application YAMLs like these three
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd     # children land in argocd's own namespace
```

Adding an app to the platform = committing one Application manifest to
`apps/`. The parent syncs it into existence; deleting the file (with a
prune policy) retires it. The failure mode you're debugging — one child
red, fleet stuck — reads identically whether children came from a parent
or a pipeline; the triage below is the same.

## Your task

1. Confirm the diagnosis on `shop-frontend` (`.status.conditions`).
2. Fix the revision:

   ```sh
   kubectl -n argocd patch application shop-frontend --type=merge \
     -p '{"spec":{"source":{"targetRevision":"HEAD"}}}'
   ```

3. Watch the fleet converge — all three Synced/Healthy:

   ```sh
   kubectl -n argocd get applications -w
   ```
