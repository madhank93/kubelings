> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern

## The situation

The `team-legacy` namespace was decommissioned ten minutes ago:

```sh
kubectl get ns team-legacy
# NAME          STATUS        AGE
# team-legacy   Terminating   87d
```

Still there. The namespace controller can't finish until **every object
inside is gone** — and it tells you exactly what's stuck:

```sh
kubectl get ns team-legacy -o jsonpath='{.status.conditions}' | python3 -m json.tool
# type: NamespaceContentRemaining
# message: 'Some resources are remaining: widgets.kubelings.dev has 1 resource instances'
```

A `Widget`? That's a custom resource:

```sh
kubectl get widgets -n team-legacy
kubectl get widget legacy-exporter -n team-legacy -o jsonpath='{.metadata.finalizers}'
# ["kubelings.dev/widget-cleanup"]
```

The Widget already has a `deletionTimestamp` — it's *trying* to die — but its
finalizer `kubelings.dev/widget-cleanup` was meant to be cleared by a
controller that was **uninstalled months ago**. The CRD survived the
uninstall; the operator pod didn't. Nothing in the cluster will ever remove
that finalizer.

In M3's PVC drill you learned finalizer-stripping is the corrupting shortcut.
This drill is the *other* case — the one legitimate use: **the controller is
provably, permanently gone.** No deployment watches Widgets; there is no
cleanup that could run. The finalizer is a promise with no one left to keep it.

## Your task

1. Confirm the diagnosis chain yourself: ns conditions → remaining resource →
   its finalizers → prove no controller owns it (what deployments/operators
   exist that would watch `widgets.kubelings.dev`?).
2. Release the orphaned finalizer:

   ```sh
   kubectl patch widget legacy-exporter -n team-legacy --type=merge \
     -p '{"metadata":{"finalizers":null}}'
   ```

3. Watch the namespace controller finish the job — `team-legacy` should be
   NotFound within seconds.

Do **not** touch the namespace's own `spec.finalizers` (the raw
`/finalize` API) — that abandons the contents instead of cleaning them up.
