---
kind: unit
title: "Kustomize: one base, many environments, zero drift"
name: kustomize-overlays-unit
---


## The situation

The audit was short and brutal: *"staging and prod are copy-pasted YAML, and
they no longer match."* Look at what prod actually runs:

```sh
kubectl -n kubelings get deploy api -o yaml | grep -E 'replicas|image:'
```

One replica (a 2 a.m. hotfix nobody reverted), a stale `nginx:1.25-alpine`,
and no `env` label — none of which is in git. This is **configuration drift**:
the cluster and the repo have quietly divorced. Every hand-`kubectl edit` is a
change with no review, no history, and no way to reproduce.

The fix isn't discipline; it's making drift *impossible to express*. Enter
**kustomize** — built into kubectl (`kubectl apply -k`), no new binary: a
**base** holds what all environments share; an **overlay** per environment
holds only the differences; nothing is copy-pasted, so nothing can disagree.

## Your task

In your lesson shell, build the tree and apply the prod overlay. Files:

```
app/
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── kustomization.yaml
└── overlays/prod/
    └── kustomization.yaml
```

`base/deployment.yaml` — the shared shape (this is the *reviewed* truth):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 1
  selector:
    matchLabels: {app: api}
  template:
    metadata:
      labels: {app: api}
    spec:
      containers:
        - name: api
          image: nginx:1.27-alpine
          resources:
            requests: {cpu: 10m, memory: 32Mi}
```

`base/service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: api
spec:
  selector: {app: api}
  ports: [{port: 80, targetPort: 80}]
```

`base/kustomization.yaml`:

```yaml
resources:
  - deployment.yaml
  - service.yaml
```

`overlays/prod/kustomization.yaml` — *only* what prod changes:

```yaml
namespace: kubelings
namePrefix: prod-
labels:
  - pairs: {env: prod}
    includeSelectors: false
    includeTemplates: true
resources:
  - ../../base
images:
  - name: nginx
    newTag: 1.27-alpine
replicas:
  - name: api
    count: 3
```

Then:

```sh
kubectl kustomize app/overlays/prod        # ALWAYS preview the build first
kubectl apply -k app/overlays/prod
kubectl -n kubelings delete deploy api     # retire the hand-drifted original
```

<details>
<summary>Hint</summary>

`kubectl kustomize <dir>` renders without applying — read it and check:
name becomes `prod-api`, replicas 3, label `env: prod` on the deployment and
pod template. If `apply -k` complains about the `labels:` field on an older
kubectl, use `commonLabels: {env: prod}` instead (it also relabels selectors —
fine for a fresh deployment, dangerous on a live one, which is exactly why the
newer `labels:` form exists).

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


## What kustomize is (and isn't)

Kustomize is **overlay-based**: plain YAML in, plain YAML out, no templates,
no variables, no logic. The kustomization file declares *transformations* —
name prefixes, labels, image pins, replica counts, strategic-merge or JSON
patches — applied over real, valid manifests you can read directly.

The other school is **template-based**: Helm. Charts with `{{ .Values.replicas }}`
holes, filled per environment, plus packaging, versioning, and a release
lifecycle (`helm install/upgrade/rollback`). Rule of thumb:

| | kustomize | Helm |
|---|---|---|
| your own apps, a few envs | ✅ natural fit | works, heavier |
| third-party software you consume | painful | ✅ that's what charts are for |
| logic/conditionals needed | not expressible (by design) | ✅ |
| learning curve | one file format | templating language + release model |

Most real clusters use **both**: Helm to install ingress-nginx and
prometheus, kustomize for the code you own. (CKA expects you to be
conversant with both; `kubectl apply -k` you've now done — Helm needs its own
binary, so here it stays theory.)

## Why drift died

The old failure: prod's YAML *was* the live object, so editing the live object
edited prod. Now prod is a **build artifact** — `base + overlay → apply`.
A hand edit to the cluster survives exactly until the next `apply -k`
regenerates the truth. Run the applies from CI (or let Argo CD/Flux do it
continuously — "GitOps" is this lesson plus a reconcile loop, M7.1's pattern
applied to *config*), and the cluster converges to git the same way a
Deployment converges to its spec.

## Prevention checklist

- `kubectl kustomize <dir>` (or `--dry-run=server -o yaml`) in code review —
  review the *rendered* output, not just the overlay.
- One overlay per environment; if two overlays keep repeating a patch, it
  belongs in the base.
- `kubectl diff -k` before `apply -k` — see drift before you overwrite it
  (that's how you *find* the 2 a.m. hotfixes worth keeping).
- Treat direct `kubectl edit/scale/set image` on managed objects as an
  incident action that must land back in git within the day.

</details>
