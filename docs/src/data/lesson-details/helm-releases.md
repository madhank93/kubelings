## The situation

Someone shipped `orders` v2 before lunch. Helm remembers everything:

```sh
helm list -n kubelings
# NAME    REVISION  STATUS   CHART         APP VERSION
# orders  2         failed   orders-0.1.0  1.0
helm history orders -n kubelings
# REVISION  STATUS      DESCRIPTION
# 1         superseded  Install complete
# 2         failed      Upgrade "orders" failed: context deadline exceeded
```

The upgrade set `replicaCount=2` and `image.tag=2.0-rc` — and `busybox:2.0-rc`
doesn't exist:

```sh
kubectl -n kubelings get pods -l app=orders-web
# orders-web-…   0/1   ImagePullBackOff
```

A Helm **release** is a versioned stack of deployed manifests: every
`install`/`upgrade`/`rollback` appends a numbered **revision** (stored as a
Secret in the release namespace — `kubectl -n kubelings get secrets -l
owner=helm`). `--wait` made Helm watch the rollout; when the new pod never
became Ready, it marked revision 2 `failed` instead of lying about success.

The chart itself is vendored on disk at `/tmp/kubelings-charts/orders` —
look at `values.yaml` and `templates/deployment.yaml` to see what the knobs
control.

## Your task

1. **Stabilize**: roll back to the last good revision:

   ```sh
   helm rollback orders 1 -n kubelings --wait
   ```

2. **Re-ship the intended change** — 2 replicas, but with a tag that exists:

   ```sh
   helm upgrade orders /tmp/kubelings-charts/orders -n kubelings \
     --set replicaCount=2 --set image.tag=1.36 --wait
   ```

3. Confirm: release `deployed`, 2/2 pods Ready, and `helm history` now reads
   like an honest changelog.

The check requires the *full* loop: a bare rollback leaves 1 replica and
fails verification — rollback is the tourniquet, not the fix.
