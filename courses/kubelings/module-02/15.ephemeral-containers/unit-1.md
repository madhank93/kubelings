---
kind: unit
title: "No shell, no exec, no problem"
name: ephemeral-containers-unit
---


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

<details>
<summary>Hint</summary>

Inside the debug container:

```sh
ls /proc/1/root/etc/app/
cat /proc/1/root/etc/app/config.json
```

(If PID 1 isn't the app, `ps` shows which one is.) The app wants
`"mode": "production"`. Then:

```sh
kubectl -n kubelings edit configmap orders-config
```

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


## Root cause

The ConfigMap `orders-config` ships `"mode": "debug"`; the app hard-requires
`"mode": "production"` and idles in a retry loop. Invisible from outside —
the only evidence is a file inside a container with no shell to read it.

## Fix

```sh
# 1. see for yourself, through the shared PID namespace:
kubectl -n kubelings debug -it orders-api --image=busybox:1.36 --target=orders-api
cat /proc/1/root/etc/app/config.json    # → "mode": "debug"
exit

# 2. fix the source:
kubectl -n kubelings patch configmap orders-config --type=merge \
  -p '{"data":{"config.json":"{\"mode\": \"production\", \"flush_interval\": 5}"}}'
```

## Verify

```sh
kubectl -n kubelings logs orders-api --tail=3    # → config ok — serving orders
kubectl -n kubelings get pod orders-api -o jsonpath='{.spec.ephemeralContainers[*].name}'
```

ConfigMap volumes update in place (kubelet sync, ≤ ~1 min) — the pod never
restarted. Note the ephemeral container stays in the spec after exit: it's
part of the pod's history now, visible to anyone auditing what was done.

## Prevention / takeaway

- `kubectl debug --target` is the front door for distroless images — keep
  images minimal *and* debuggable.
- `--target` shares the PID namespace; `/proc/<pid>/root/` is the app's
  filesystem. Without `--target` you get a sidecar-like container that can
  see volumes only if you copy the pod (`--copy-to`).
- Env-var config wouldn't have healed in place — mounted ConfigMaps reload,
  env vars don't (M3's `pattern-secret-not-reloaded` drills that trap).

</details>
