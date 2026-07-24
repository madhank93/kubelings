---
kind: unit
title: "App of apps: one bad child in the fleet"
name: gitops-argocd-appofapps-unit
---


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

<details>
<summary>Hint</summary>

`HEAD` follows the repo's default branch; a real platform pins children to
tags or SHAs (that's the point of `targetRevision`) — but the pin must
name something that exists. `git ls-remote <repo>` lists what does.

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


## Fix

```sh
kubectl -n argocd patch application shop-frontend --type=merge \
  -p '{"spec":{"source":{"targetRevision":"HEAD"}}}'
```

## Fleet triage, the repeatable order

1. `kubectl -n argocd get applications` — the one-line fleet view; find
   the non-green rows.
2. Per red app: `.status.conditions` (can it compare?) → `.status.sync`
   (does live match git?) → `.status.health` (is it up?). Same ladder as
   10.1, applied N times.
3. Revision errors (`unknown revision`), path errors (`app path does not
   exist`), and credential errors (`authentication required`) are the
   three ComparisonError classics — all fixed in the child's `spec.source`.

## Prevention / takeaway

- **Sync waves** encode dependency order declaratively; combine with the
  parent pattern and "bring up the whole platform in order" is one sync.
  (Resource-level hooks/waves exist too — same annotation on any object.)
- Pin children to tags/SHAs, not branches — but validate pins in CI
  (`git ls-remote` is a one-line check) so a typo'd revision dies in
  review, not on the fleet dashboard.
- The parent pattern means *Application YAML lives in git too* — the
  platform's shape gets code review, history, and rollback like everything
  else. That's the actual product of GitOps: the audit trail.
- ApplicationSets are the industrial version (generate children from
  cluster lists, monorepos, PR previews) — same status ladder when they
  break.

</details>
