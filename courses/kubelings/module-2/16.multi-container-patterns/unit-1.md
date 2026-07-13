---
kind: unit
title: "Three pods, three broken patterns"
name: multi-container-patterns-unit
---


## The situation

A pod is not "a container" — it's a set of containers sharing network and
volumes, and that sharing is exactly what the three classic multi-container
patterns exploit:

- **Sidecar** — a helper alongside the app: here, `log-forwarder` tails the
  app's log file off a shared `emptyDir` and ships it.
- **Ambassador** — a local proxy: the app talks to `localhost:8000`, the
  ambassador forwards to the real `payments` Service. App code stays dumb;
  routing is the ambassador's problem.
- **Adapter** — a translator: the app emits legacy-format metrics, the adapter
  rewrites them into Prometheus format where the scraper expects them.

Three pods are running in `kubelings` — one per pattern, each with one bug:

```sh
kubectl -n kubelings get pods
kubectl -n kubelings logs orders-logs -c log-forwarder     # …nothing useful
kubectl -n kubelings exec orders-checkout -c app -- wget -qO- -T2 http://127.0.0.1:8000   # hangs/fails
kubectl -n kubelings exec orders-metrics -c app -- ls /shared/                             # no metrics.prom
```

## Your task

1. **`orders-logs`** (sidecar): the app writes `order shipped` lines every 2 s,
   but the forwarder ships nothing. Compare *where the app writes* with *where
   the sidecar reads*. Fix the forwarder.
2. **`orders-checkout`** (ambassador): the app expects `localhost:8000` to
   reach the `payments` Service. Check what the ambassador actually forwards
   to (`kubectl get pod orders-checkout -o yaml`), and what port `payments`
   really serves on.
3. **`orders-metrics`** (adapter): the adapter converts the legacy metrics
   correctly — check its stdout — but the scraper reads
   `/shared/metrics.prom`. Route the converted output there.

Container `command`/`args` and volume mounts are **immutable on a running
pod** — the fix loop is: dump, edit, replace:

```sh
kubectl -n kubelings get pod <name> -o yaml > /tmp/<name>.yaml
# edit, then:
kubectl -n kubelings replace --force -f /tmp/<name>.yaml
```

<details>
<summary>Hint</summary>

- Sidecar: both containers mount *something* at `/var/log/app` — but check
  `volumeMounts` against `volumes`: the app writes into volume `logs`, the
  forwarder mounted volume `scratch`. Same path, different disk. Point the
  forwarder's mount at `logs`.
- Ambassador: `TCP:payments:9999` → `TCP:payments:80`.
- Adapter: it `print`s to stdout. Redirect: `... > /shared/metrics.prom`.

</details>

::simple-task
---
:tasks: tasks
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>


## Root cause, per pod

| Pod | Pattern | Bug |
|---|---|---|
| `orders-logs` | sidecar | app writes `app.log` into volume `logs`; the forwarder mounts volume `scratch` at the same path and tails its own empty decoy file |
| `orders-checkout` | ambassador | socat forwards `localhost:8000 → payments:9999`; the Service serves on `80` |
| `orders-metrics` | adapter | the awk conversion is correct but goes to stdout; the contract is the shared file `/shared/metrics.prom` |

## Fix

Dump each pod, apply the one-line change, `kubectl replace --force`:

- `orders-logs` — the paths match, the *volumes* don't. This is the classic
  form of the bug precisely because `ls /var/log/app/` shows a plausible
  `app.log` in both containers: two different emptyDirs, two different files.
  One-word fix in the forwarder's mount:

  ```yaml
  - name: log-forwarder
    command: ["sh", "-c", "tail -F /var/log/app/app.log"]
    volumeMounts:
      - {name: logs, mountPath: /var/log/app}   # was: name: scratch
  ```

- `orders-checkout` ambassador args:

  ```yaml
  args: ["TCP-LISTEN:8000,fork,reuseaddr", "TCP:payments:80"]
  ```

- `orders-metrics` adapter command — same pipeline, redirected:

  ```yaml
  command: ["sh", "-c", "while true; do if [ -f /shared/metrics.log ]; then awk -F'|' '{print \"orders_processed_total \" $3}' /shared/metrics.log | tail -1 > /shared/metrics.prom; fi; sleep 2; done"]
  ```

## Verify

```sh
kubectl -n kubelings logs orders-logs -c log-forwarder --tail=3        # order shipped …
kubectl -n kubelings exec orders-checkout -c app -- wget -qO- http://127.0.0.1:8000 | head -1
kubectl -n kubelings exec orders-metrics -c app -- cat /shared/metrics.prom
```

## Prevention / takeaway

- Shared `emptyDir` is a *contract*: both containers must reference the same
  **volume name**, not just the same mount path. Identical paths over
  different volumes is the invisible version of this bug — `ls` looks right
  in both containers.
- Ambassador upstreams belong in config, not baked into args — when the
  Service port changes, nobody remembers the socat line.
- An adapter's output location is its entire job. stdout is where debugging
  output goes to feel productive.
- Sidecars became first-class in K8s 1.28+ (`initContainers` with
  `restartPolicy: Always`) — they start before and stop after the app.

</details>
