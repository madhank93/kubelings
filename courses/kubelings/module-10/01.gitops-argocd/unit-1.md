---
kind: unit
title: "Argo CD: the app that refuses to sync"
name: gitops-argocd-unit
---


## The situation

Welcome to platform engineering: from here on, nobody `kubectl apply`s
application manifests by hand. **GitOps** inverts the deploy arrow — instead
of CI *pushing* YAML at the cluster, an in-cluster agent *pulls* the desired
state from git and continuously reconciles toward it. Git becomes the
database of intent (a very M7.1 idea: reconcile loops, one level up), and a
deploy is a commit.

**Argo CD** is that agent here, freshly installed in the `argocd` namespace:

```sh
kubectl -n argocd get pods
# argocd-application-controller-0   ← the reconciler
# argocd-repo-server-…              ← clones repos, renders manifests
# argocd-server-…                   ← API/UI
```

One **Application** — Argo CD's CRD binding a git source to a cluster
destination — is supposed to deploy the classic guestbook:

```sh
kubectl -n argocd get application guestbook
# NAME        SYNC STATUS   HEALTH STATUS
# guestbook   Unknown       Healthy
```

`Unknown` is not a phase to wait out. Ask why:

```sh
kubectl -n argocd get application guestbook -o jsonpath='{.status.conditions}' | python3 -m json.tool
# "type": "ComparisonError"
# "message": "… app path does not exist: apps/guestbook …"
```

The Application's `spec.source.path` says `apps/guestbook`; the repo's
guestbook actually lives at `guestbook`. Argo CD can't render the desired
state, so it can't even *compare* — sync status `Unknown`, forever, while
the UI shows a comforting green Healthy for the zero resources it manages.

## Your task

1. Confirm the real path yourself — the repo is public:
   `https://github.com/argoproj/argocd-example-apps`, directory `guestbook`.
2. Fix the Application's source path:

   ```sh
   kubectl -n argocd patch application guestbook --type=merge \
     -p '{"spec":{"source":{"path":"guestbook"}}}'
   ```

3. Auto-sync is on (`syncPolicy.automated`) — watch the machine do the rest:

   ```sh
   kubectl -n argocd get application guestbook -w
   # Unknown → OutOfSync → Synced   /   Healthy
   kubectl -n kubelings get deploy guestbook-ui
   ```

<details>
<summary>Hint</summary>

The status field trio to read on any stuck Application, in order:

```sh
kubectl -n argocd get app guestbook -o jsonpath='{.status.conditions}'      # can it compare?
kubectl -n argocd get app guestbook -o jsonpath='{.status.sync.status}'     # does live match git?
kubectl -n argocd get app guestbook -o jsonpath='{.status.health.status}'   # is the workload up?
```

Comparison errors always win — sync and health mean nothing until
conditions are clean.

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
kubectl -n argocd patch application guestbook --type=merge \
  -p '{"spec":{"source":{"path":"guestbook"}}}'
kubectl -n argocd get application guestbook -w   # → Synced / Healthy
```

## The two status axes (and the third that gates them)

- **Sync**: does the live state match what git renders? (`Synced` /
  `OutOfSync`)
- **Health**: is the workload actually working? (Deployment available,
  pods Ready — Argo CD has per-kind health checks)
- **Conditions**: can Argo CD do its job at all? `ComparisonError` means
  the source is unreadable — wrong path, bad revision, private repo
  without credentials. A comparison error makes both other columns lies.

Synced-but-Degraded and OutOfSync-but-Healthy are both normal, meaningful
states — the pair is exactly the "desired vs observed" split every
controller has (M7.1), surfaced as UX.

## Prevention / takeaway

- `spec.syncPolicy.automated.selfHeal: true` means manual `kubectl` edits
  to managed resources get reverted within seconds — the cluster's truth
  is git, deliberately. Drift becomes visible (`OutOfSync`) instead of
  permanent (the Weaveworks-era GitOps pitch in one flag).
- No `argocd` CLI was needed: the Application CR *is* the interface, and
  jsonpath on `.status` is how you script against it — same skill as every
  other controller this course taught.
- Path/revision typos are the #1 first-day Argo CD failure; the fix
  discipline is reading `.status.conditions` before anything else.
- One Application is a pet. The pattern scales as app-of-apps — next
  lesson.

</details>
