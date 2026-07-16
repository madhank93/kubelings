## The situation

A nightly **`data-import`** Job in the `kubelings` namespace never finishes. The
downstream step waits for `kubectl wait --for=condition=complete job/data-import`
and hangs forever. The pod looks "Running" indefinitely.

A Job is **complete only when its container exits 0**. This one runs
`sleep infinity` — it never exits, so the Job stays active forever.

## Your task

Make `data-import` **run to completion** (a Job named `data-import` in `kubelings`
with `status.succeeded ≥ 1` and condition `Complete=True`).

```sh
kubectl -n kubelings get job data-import -o yaml | less
kubectl -n kubelings logs job/data-import
```

> Job pod templates are immutable — to change the command you must delete and
> recreate the Job.
