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
