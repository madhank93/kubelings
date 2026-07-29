---
kind: unit
title: "Incident replay — a couple of characters pruned the world (Skyscanner)"
name: incident-gitops-prune-unit
---


## The real incident

**Skyscanner**, 24 August 2021. A change described internally as a no-op edited a
configuration file at the root of the "cells" tree — the file that carries
regional configuration to every cell, everywhere. Two characters went missing:
the template's `{{ }}` delimiters.

Source: [Skyscanner — How a couple of characters brought down our site](https://medium.com/@SkyscannerEng/how-a-couple-of-characters-brought-down-our-site-356ccaf1fbc3)

Without the delimiters the templating system produced nothing usable. That file
was rolled out **globally and immediately** — the very property that made it
useful for regional config. Argo CD then did its one job: it compared desired
state (now: no valid namespaces, no services) with live state, and reconciled the
difference by **deleting 478 services across every namespace, availability zone
and region on earth**. skyscanner.net and the mobile apps were down for four and
a half hours; full recovery across all regions took two more days.

The Cells Architecture existed precisely to contain regional failure. It could
not contain a *central config* failure, because every cell was faithfully
obeying the same corrupted instruction. **Automation does not distinguish "you
asked for nothing" from "you asked me to delete everything."**

## This cluster, right now

`kubelings` runs the same shape, three cells small: a reconciler script renders
one Service per region from `cells-template`, then applies it with `--prune`, so
anything it manages but no longer renders is removed.

The regions line already lost its delimiters, and the reconciler has already run:

```sh
kubectl -n kubelings get svc -l app.kubernetes.io/managed-by=cells   # nothing left
kubectl -n kubelings get configmap cells-config -o jsonpath='{.data.regions}{"\n"}'
kubectl -n kubelings get configmap cells-reconciler -o jsonpath='{.data.run\.sh}'
```

Running a reconcile is how the "agent" is invoked here — it reads the script out
of the ConfigMap, so your edits take effect on the next run:

```sh
kubectl -n kubelings get configmap cells-reconciler -o jsonpath='{.data.run\.sh}' > /tmp/run.sh
NS=kubelings CONFIG_CM=cells-config sh /tmp/run.sh
```

<!-- d2:prune -->
```text
 ┌───────────────────┐
 │config loses {{ }} │
 │                   │
 └───────────────────┘
         │            
rolled out globally   
         │            
         ▼            
 ┌────────────────┐   
 │render is empty │   
 │                │   
 └────────────────┘   
         │            
   apply --prune      
         │            
         ▼            
  ┌─────────────────┐ 
  │all cells pruned │ 
  │                 │ 
  └─────────────────┘ 
```
<!-- /d2:prune -->

## Your task

Bring the site back, then make this class of change survivable:

1. Restore the region list in `cells-config` — the three cells are `eu`, `us`,
   `ap` — and reconcile so `web-eu`, `web-us`, `web-ap` exist again.
2. Make the reconciler **fail closed**: patch `run.sh` so a render missing its
   regions **refuses to apply at all** rather than pruning. Refuse *before* the
   apply, not after.
3. Leave the reconciler's `--prune` in place. Pruning is not the bug — pruning
   an empty render is.

The check proves the guard by running the reconciler twice: once with the good
config (must succeed, cells intact) and once against a frozen copy of the broken
one, `cells-config-broken` (must fail, cells intact). A guard that refuses
*everything* is not a guard — it's an outage with better manners.

<details>
<summary>Hint</summary>

Restore, reconcile, then guard:

```sh
kubectl -n kubelings patch configmap cells-config --type=merge \
  -p '{"data":{"regions":"eu us ap"}}'
```

Add the sanity check to `run.sh` immediately before the `kubectl apply` line —
edit it in place:

```sh
kubectl -n kubelings edit configmap cells-reconciler
```

```sh
# fail closed: an empty/short render is a bug, never a deletion order
want=$(printf '%s\n' $regions | grep -c . || true)
have=$(grep -c '^kind: Service' "$rendered" || true)
if [ "$want" -lt 3 ] || [ "$have" -lt "$want" ]; then
  echo "refusing to reconcile: rendered $have Services for $want regions" >&2
  exit 1
fi
```

Then reconcile again and confirm both halves of the contract — the good config
still applies, the broken one refuses:

```sh
kubectl -n kubelings get configmap cells-reconciler -o jsonpath='{.data.run\.sh}' > /tmp/run.sh
NS=kubelings CONFIG_CM=cells-config        sh /tmp/run.sh   # must succeed
NS=kubelings CONFIG_CM=cells-config-broken sh /tmp/run.sh   # must refuse, delete nothing
kubectl -n kubelings get svc -l app.kubernetes.io/managed-by=cells
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

## Root cause chain

1. **A rarely-changed file at the top of the tree.** Blast radius is a property
   of *position*, not of diff size. The smallest edit to the most central file is
   the most dangerous change you can make.
2. **The renderer failed soft.** Missing delimiters produced output that was
   syntactically fine and semantically empty. Nothing rejected it.
3. **The reconciler treated empty as intentional.** Desired state said "no
   services"; prune deleted 478 of them, worldwide, in the time it takes to sync.
4. **Global-and-immediate rollout.** No canary region, no soak. The mechanism
   that makes central config convenient is the same one that makes it fatal.

## The fix, generalized (Skyscanner's own conclusions)

- **Config is code: lint it, test it, review it.** A schema/render test in CI
  catches "renders to nothing" before a human has to notice it at 3am.
- **Fail closed on empty desired state.** Every reconciler that can delete needs
  a floor: "if the render produces fewer than N objects, stop and page." Argo CD
  ships exactly this — `spec.syncPolicy.automated.prune: false`, plus
  `PruneLast`, `PrunePropagationPolicy`, and the `FailOnSharedResource` /
  `allowEmpty: false` guards. `allowEmpty: false` *is* this lab's lesson,
  upstream: an Application that renders no resources refuses to sync.
- **Never roll the world at once.** Regional config should roll region by region
  with a bake time, so the second cell tells you what the first one learned.
- **Rehearse restore.** Skyscanner's recovery was slower than the failure by two
  orders of magnitude, and that gap was procedural, not technical.

## Ecosystem equivalents worth wiring up today

| Tool | The guard |
|---|---|
| Argo CD | `automated.prune`, `allowEmpty: false`, sync windows, per-app project restrictions |
| Flux | `spec.prune`, `--no-remediation` on a failed build, `Kustomization` health checks |
| `kubectl apply --prune` | `--prune-allowlist` (never prune a type you didn't intend to own) + your own render-size check |
| Helm | `--atomic` rollback, `lookup`-free templates, `helm template \| kubeconform` in CI |

## Prevention

```sh
# Which Argo apps can delete for you today?
kubectl get applications -A -o json | jq -r '.items[]
  | select(.spec.syncPolicy.automated.prune == true)
  | "\(.metadata.name)  allowEmpty=\(.spec.syncPolicy.automated.allowEmpty // false)"'
```

Any row with `prune=true` and `allowEmpty=true` is one bad render from this
incident. Related war stories: M9 `incident-spotify-delete` (the same blast radius
via Terraform state) and M6 `incident-webhook-outage` (an addon whose failure mode
was "deny everything").

</details>
