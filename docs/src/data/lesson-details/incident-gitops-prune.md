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

<figure class="lesson-diagram">
<img src="/diagrams/incident-gitops-prune-prune.svg" alt="incident-gitops-prune prune diagram" loading="lazy">
</figure>

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
