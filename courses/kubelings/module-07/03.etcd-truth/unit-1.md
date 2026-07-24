---
kind: unit
title: "etcd: the one source of truth"
name: etcd-truth-unit
---


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

<details>
<summary>Hint</summary>

etcd runs as a static pod on the control-plane node. `etcdctl` is inside its
container; the client certs are on the node. On this kind cluster:

```sh
kubectl -n kube-system exec -it etcd-$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}' | sed 's/.*/&/;s/^/kubelings-control-plane/' 2>/dev/null || echo kubelings-control-plane) -- sh -c '
  ETCDCTL_API=3 etcdctl \
    --cacert /etc/kubernetes/pki/etcd/ca.crt \
    --cert /etc/kubernetes/pki/etcd/server.crt \
    --key /etc/kubernetes/pki/etcd/server.key \
    get /registry/configmaps/kubelings/etcd-proof --print-value-only'
```

You'll see your `i-was-here` marker in the raw stored object. (The exact etcd pod
name is `etcd-<control-plane-node>`; `kubectl -n kube-system get pod | grep etcd`
if the one-liner's guess is off.)

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


## What you just saw

Your ConfigMap stored at the key `/registry/configmaps/kubelings/etcd-proof`.
That path *is* the object's identity: `/registry/<resource>/<namespace>/<name>`.
`kubectl get` is sugar over "API server does a range read on `/registry/...` in
etcd, filters by RBAC, returns JSON."

## Why the whole architecture falls out of this

- **Watches** (next-level machinery): controllers don't poll — they open a watch
  on the API server, which relays etcd's change stream. Delete a pod → etcd key
  removed → watch event fires → ReplicaSet controller reconciles. The loop from
  the last lesson is powered by etcd's revision-ordered event log.
- **`resourceVersion`** on every object is literally etcd's revision. Optimistic
  concurrency (`kubectl apply` conflict errors, `409 Conflict`) is etcd's
  compare-and-swap surfacing through the API.
- **Consistency & HA:** etcd is a Raft cluster — needs a quorum (majority) to
  accept writes. That's why control planes come in odd numbers (3, 5): to
  tolerate 1 or 2 failures. Lose quorum and the cluster goes **read-only** — a
  real and scary failure mode (several Incident Library entries are etcd quorum
  loss).

## Operational truths this hands you

- **Backups:** `etcdctl snapshot save` is your disaster-recovery lifeline. No
  etcd backup = no cluster restore. Schedule it; test the restore.
- **etcd is precious and small:** keep it on fast disks, watch its DB size,
  don't stuff giant objects/Secrets in — every write goes through Raft consensus.
- **Encrypt Secrets at rest:** by default, Secret values sit in etcd
  base64-encoded, *not encrypted*. Anyone who reads etcd (or a backup) reads
  your Secrets. `EncryptionConfiguration` fixes it — and now you understand
  exactly why it matters (you just read a value straight out of the store).

## The one-sentence version

**Kubernetes is a set of controllers reconciling the world toward desired state,
and that desired state is rows in etcd behind a stateless REST API.** Every other
lesson in this course is a corollary.

</details>
