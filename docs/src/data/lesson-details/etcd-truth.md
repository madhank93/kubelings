## The situation

When you run `kubectl get pods`, where does the answer *come from*? Not from the
nodes. Not from the kubelet. The API server reads it from **etcd** — a
distributed key-value store that holds the entire cluster state as keys under
`/registry/...`. Every Deployment, Secret, Node, RBAC binding you've created in
this course is a value in there right now.

This single fact explains a lot:

- The **API server is stateless** — it's a validating, authorizing REST facade
  over etcd. Kill it, start another; nothing is lost because it stores nothing.
- **"Back up etcd" = "back up the cluster."** Lose etcd, lose every object.
  (The nodes keep running their current containers, but the control plane has
  amnesia — no desired state to reconcile toward.)
- **Reconciliation reads desired state *from etcd*** (via the API server). The
  loop you saw last lesson closes the gap between etcd's spec and the world.

Let's see it with your own eyes.

## Your task

1. Create a ConfigMap that will be your sentinel:

   ```sh
   kubectl -n kubelings create configmap etcd-proof --from-literal=marker=i-was-here
   ```

2. Then go find it *inside etcd* — read the raw key (see the hint). The check
   only requires the ConfigMap to exist, but the learning is in step 2: seeing
   your object as a `/registry` key.
