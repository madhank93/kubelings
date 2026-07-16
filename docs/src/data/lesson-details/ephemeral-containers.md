## The situation

`orders-api` is up — `Running`, `1/1 Ready` — but it isn't serving anything.
The logs only say it refuses to run:

```
fatal: config invalid, refusing to serve (retrying)
```

*Which* config? *Invalid how?* You reach for the usual door:

```
$ kubectl -n kubelings exec -it orders-api -- sh
error: Internal error occurred: ... "sh": executable file not found
```

There is no `sh`. No `cat`, no `ls`, no anything — the image is **distroless**:
the app binary and its runtime, nothing else. That's great for supply-chain
hygiene (M6.6 made you strip images for exactly this reason) and terrible for
2am debugging — until you know about **ephemeral containers**. `kubectl debug`
grafts a *temporary* container with real tools into the running pod. With
`--target`, it shares the app's **PID namespace**, and through
`/proc/<pid>/root/` you can read the app container's entire filesystem —
mounts, config files, all of it. No restart, no image rebuild.

## Your task

1. Attach a debug container to `orders-api`:

   ```sh
   kubectl -n kubelings debug -it orders-api --image=busybox:1.36 --target=orders-api
   ```

2. From inside it, find the app process and inspect the config file it reads
   (the app's filesystem hangs off `/proc/1/root/...`).
3. The config lives in a ConfigMap. Fix the bad value, wait for the kubelet to
   project the update (up to ~1 minute), and confirm the logs flip to
   `config ok — serving orders`.

```sh
kubectl -n kubelings get pod orders-api -o wide
kubectl -n kubelings logs orders-api --tail=5
```
