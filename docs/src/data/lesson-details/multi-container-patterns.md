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
