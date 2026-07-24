---
kind: unit
title: "Drill — the namespace stuck Terminating"
name: pattern-namespace-terminating-unit
---


> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern
> write-up: [Pattern: namespace stuck Terminating](https://kubelings.madhan.app/incidents/pattern-namespace-terminating/).

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

<details>
<summary>Hint</summary>

The full trail, in order:

```sh
kubectl get ns team-legacy -o jsonpath='{.status.conditions[?(@.type=="NamespaceContentRemaining")].message}'
kubectl api-resources --verbs=list --namespaced -o name | head -50   # widgets.kubelings.dev is in there
kubectl get widgets -n team-legacy
kubectl patch widget legacy-exporter -n team-legacy --type=merge -p '{"metadata":{"finalizers":null}}'
kubectl get ns team-legacy   # NotFound
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


## The pattern (why this recurs everywhere)

Operator uninstalls routinely go in the wrong order: `helm uninstall` (or
`kubectl delete -f operator.yaml`) removes the *controller* first, leaving
CRDs and custom resources behind — each still carrying finalizers only that
controller could clear. The debt is invisible until someone deletes a
namespace, and every namespaced custom resource with an orphaned finalizer
becomes a roadblock. Cluster teardown scripts hit this so often that "kubectl
delete ns hangs" is a rite of passage.

## Fix

```sh
kubectl patch widget legacy-exporter -n team-legacy --type=merge \
  -p '{"metadata":{"finalizers":null}}'
# the namespace controller notices within seconds:
kubectl get ns team-legacy    # Error from server (NotFound)
```

## The decision rule for finalizer-stripping

| Situation | Action |
|---|---|
| Controller alive, finalizer pending | **wait / fix the controller** — stripping corrupts (the PVC drill) |
| Controller permanently gone (uninstalled operator, dead CRD) | strip — nothing else can ever clear it (this drill) |
| Not sure | find out first: `kubectl get deploy,sts -A \| grep <operator>`; check the CRD's docs |

The two drills are the same command with opposite verdicts — the difference
is whether anyone is left to keep the promise.

## Prevention / takeaway

- **Uninstall order**: custom resources → wait for cleanup → CRDs →
  controller. The controller dies *last*, after its finalizers are done.
- Namespace `status.conditions` names the blocker every time — read it
  before trying anything (`NamespaceContentRemaining`,
  `NamespaceFinalizersRemaining`).
- Audit for orphans: any CRD without a running controller is future debt —
  `kubectl get crds` vs. what your operators actually watch.
- Never clear the *namespace's* `spec.finalizers` via the `/finalize` API to
  "unstick" it — that orphans the remaining objects in etcd instead of
  deleting them.

</details>
