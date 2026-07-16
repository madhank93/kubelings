## The situation

Init dropped two kubeconfig files in `/tmp/kubelings-kubeconfig/`:

```sh
DIR=/tmp/kubelings-kubeconfig
kubectl config get-contexts --kubeconfig $DIR/main
# CURRENT   NAME      CLUSTER   NAMESPACE
# *         prod      …         default
#           staging   …         kubelings
#           dev       …         dev
kubectl config get-contexts --kubeconfig $DIR/extra
#           observability   …   monitoring
```

A **context** is just a named triple: *cluster* (server + CA) + *user*
(credentials) + default *namespace*. The file's `current-context` decides
where every bare `kubectl` command lands — and right now it's `prod`. Every
`kubectl delete` you type without `--context` goes to production. This exact
default has starred in several of this course's incident files (Spotify's
cluster deletion began with a terminal pointed at the wrong cluster).

## Your task

All against the files in `$DIR` (leave your real `~/.kube/config` alone):

1. **Switch off prod** — make `staging` the current context of `$DIR/main`:

   ```sh
   kubectl config use-context staging --kubeconfig $DIR/main
   ```

2. **Merge** the teammate's `extra` file into a single self-contained file at
   `$DIR/merged`, keeping all four contexts. The merge operator is the
   `KUBECONFIG` path list plus `--flatten`:

   ```sh
   KUBECONFIG=$DIR/main:$DIR/extra kubectl config view --flatten > $DIR/merged
   ```

3. Prove the merged file works:

   ```sh
   kubectl --kubeconfig $DIR/merged config get-contexts
   kubectl --kubeconfig $DIR/merged --context=staging get pods
   ```
