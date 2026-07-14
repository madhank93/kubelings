---
title: "Pattern: namespace stuck Terminating"
description: "[PATTERN] Synthetic composite — a namespace won't finish deleting because a custom resource carries a finalizer whose controller was uninstalled long ago."
---

> **[PATTERN] scenario** — a synthetic composite of a failure mode reported
> across many production clusters. **No specific company**; details are
> representative, not cited. (Real, cited incidents are marked `[REAL]` in the
> [Incident Library](/reference/incident-library/).)

## Situation

A decommissioned namespace sits in `Terminating` for ten minutes, an hour, a
day. Teardown pipelines time out; someone finds a StackOverflow snippet that
force-clears the namespace's finalizers through the raw `/finalize` API; the
namespace vanishes and everyone moves on — with orphaned objects now stranded
in etcd.

## Root cause

The namespace controller cannot finish until **every object inside the
namespace is deleted**. Some object — almost always a *custom resource* —
carries a finalizer whose controller no longer exists: the operator was
uninstalled (controller first, CRDs and CRs left behind), and its finalizer
is now a promise nobody is left to keep. The object has a `deletionTimestamp`
and waits forever; the namespace waits on the object.

## Diagnosis

The namespace names its blocker — read the conditions before anything else:

```sh
kubectl get ns <name> -o jsonpath='{.status.conditions}'
# NamespaceContentRemaining: "widgets.example.com has 1 resource instances"

kubectl get widgets -n <name>
kubectl get widget <obj> -n <name> -o jsonpath='{.metadata.finalizers}'
# is anything still watching this CRD?
kubectl get deploy,sts -A | grep -i <operator>
```

## Fix

Only after proving the controller is permanently gone, release the orphaned
finalizer **on the blocking object** (not on the namespace):

```sh
kubectl patch widget <obj> -n <name> --type=merge \
  -p '{"metadata":{"finalizers":null}}'
# the namespace controller finishes within seconds
```

This is the one legitimate finalizer-strip: contrast with
[Pattern: PVC stuck Terminating](/incidents/pattern-pvc-terminating/), where
the controller is alive and stripping corrupts. Same command, opposite
verdicts — the difference is whether anyone is left to keep the promise.

**Anti-pattern:** clearing the *namespace's* own `spec.finalizers` via the
raw `/finalize` API. The namespace disappears but its remaining objects are
abandoned in etcd, still consuming storage and still returned by
cluster-scoped LISTs.

## Prevention

- Uninstall order for operators: custom resources → wait for their cleanup →
  CRDs → controller last.
- Audit for debt: every CRD without a running controller is a future stuck
  namespace.
- Alert on namespaces in `Terminating` beyond a threshold; the conditions
  field makes the alert actionable.

## What it teaches

| Concept | Kubelings module |
|---|---|
| Finalizers, two-phase deletion, namespace lifecycle | M3 (`pattern-namespace-terminating`) |
| CRD/operator lifecycle | M7 Internals (`crd-operators`, `build-an-operator`) |
