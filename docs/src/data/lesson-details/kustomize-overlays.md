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
