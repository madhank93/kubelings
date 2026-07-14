---
kind: unit
title: "Crossplane: the composition missing its provider"
name: crossplane-compositions-unit
---


## The situation

The platform team's promise: developers don't file tickets for databases —
they apply an `XDatabase` and infrastructure appears. **Crossplane** is
the machinery behind that promise, and it's three CRDs deep:

- **XRD** (`CompositeResourceDefinition`) — *defines the API*: teaches the
  cluster the `XDatabase` kind, its schema, its defaults. It's M7.5's CRD
  lesson, generated from a higher-level definition.
- **Composition** — *implements the API*: when an XDatabase appears, run
  this pipeline of functions to render the real resources (cloud
  databases, buckets, here a stand-in `NopResource`).
- **Provider / Function** (packages) — *the workers*: a Provider is an
  operator that reconciles some API group's resources against the real
  world; a Function renders compositions.

A developer claimed one. It's not going well:

```sh
kubectl -n kubelings get xdatabase orders-db
# NAME        SYNCED   READY
# orders-db   False    …
kubectl -n kubelings get xdatabase orders-db -o jsonpath='{.status.conditions[?(@.type=="Synced")].message}'
# cannot compose resources: … no matches for kind "NopResource"
#   in version "nop.crossplane.io/v1alpha1"
```

You've read that error class before — M3's namespace drill (a kind with no
CRD), M6.11's Rego path (a reference into nothing). The Composition's
pipeline emits `nop.crossplane.io/v1alpha1 NopResource`, and **nothing
installed serves that API**: the platform team shipped the recipe but not
the worker. Check what packages exist:

```sh
kubectl get providers.pkg.crossplane.io,functions.pkg.crossplane.io
# NAME: function-patch-and-transform   INSTALLED  HEALTHY   ← the renderer
# (no providers)                                            ← the gap
```

## Your task

1. Install the provider that owns `nop.crossplane.io` — pinned, like every
   package in this course:

   ```sh
   kubectl apply -f - <<'EOF'
   apiVersion: pkg.crossplane.io/v1
   kind: Provider
   metadata:
     name: provider-nop
   spec:
     package: xpkg.crossplane.io/crossplane-contrib/provider-nop:v0.5.0
   EOF
   ```

2. Watch the package machinery: `kubectl get providers.pkg.crossplane.io -w` until
   `INSTALLED/HEALTHY`, then the claim heal itself:

   ```sh
   kubectl -n kubelings get xdatabase orders-db -w
   # SYNCED True → READY True (the NopResource "provisions" in ~5s)
   kubectl -n kubelings get nopresources
   ```

<details>
<summary>Hint</summary>

Nothing to retry by hand: Crossplane's reconcile loops (M7.1, always M7.1)
pick up the new CRDs and re-render the stuck composition on their own
within a minute. If SYNCED stays False, re-read the condition message —
it updates with the *current* blocker.

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


## Fix

```sh
kubectl apply -f - <<'EOF'
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-nop
spec:
  package: xpkg.crossplane.io/crossplane-contrib/provider-nop:v0.5.0
EOF
```

The Provider package installs its CRDs (`NopResource` among them) and
starts its controller; the composition's next reconcile finds the kind,
renders, and the claim converges to Ready.

## Root cause, platform edition

A Composition is a *contract between recipe and workers*: every `base:`
kind it emits must have an installed, healthy package serving it. The
dependency is invisible in the YAML — nothing in a Composition declares
"requires provider-nop" (real packages can declare dependencies in their
`crossplane.yaml`; hand-applied stacks like this one carry the dependency
in someone's head). Environments drift apart exactly here: staging has the
provider, the new cluster doesn't, and every claim arrives broken.

## Why platform teams reach for this

- **The XRD is an API boundary**: developers see `XDatabase` with a `size`
  field — not 400 lines of cloud-provider YAML, not credentials (provider
  configs stay with the platform team — the M6 secrets discipline holds).
- **provider-nop is not a toy detail**: it "provisions" nothing on a
  timer, which makes it the standard way to test XRDs/Compositions in CI
  before pointing them at money. Swap the `base:` kind for
  `provider-kubernetes` or a cloud provider's and the lesson is identical.
- Crossplane vs CAPI (10.5): CAPI reconciles *clusters*; Crossplane
  reconciles *everything else with an API* — same operator pattern, wider
  aim. The full platform stack you now hold: GitOps delivers YAML
  (10.1–10.3), tenants bound teams (10.4), CAPI mints clusters (10.5),
  Crossplane serves infrastructure as self-service APIs.

</details>
