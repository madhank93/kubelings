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
