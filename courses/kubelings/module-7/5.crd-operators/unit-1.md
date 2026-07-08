---
kind: unit
title: "CRDs: teach the API server a new noun"
name: crd-operators-unit
---


## The situation

The platform team's new backup tooling ships this manifest, and asks you to
apply it:

```yaml
apiVersion: kubelings.dev/v1
kind: BackupSchedule
metadata:
  name: nightly-etcd
  namespace: kubelings
spec:
  schedule: "0 2 * * *"
  target: etcd
  retainDays: 14
```

```sh
kubectl apply -n kubelings -f backupschedule.yaml
```

```
error: unable to recognize: no matches for kind "BackupSchedule"
  in version "kubelings.dev/v1"
```

Nothing is *broken*. `Deployment`, `Pod`, `Service` — every kind you've used
is an entry in the API server's schema; `BackupSchedule` simply isn't one of
them **yet**. The fix is the single most consequential design decision in
Kubernetes: the API is *extensible at runtime*. A
**CustomResourceDefinition** is itself just an object you apply — and the API
server responds by growing a whole new REST endpoint, with storage in etcd
(7.3), RBAC verbs (6.1), and watch support (7.1), for free.

## Your task

1. Define the noun — apply this CRD:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  # MUST be <plural>.<group> — the API server enforces the naming contract
  name: backupschedules.kubelings.dev
spec:
  group: kubelings.dev
  scope: Namespaced
  names:
    kind: BackupSchedule
    singular: backupschedule
    plural: backupschedules
    shortNames: [bks]
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required: [schedule, target]
              properties:
                schedule: {type: string}
                target: {type: string, enum: [etcd, volumes]}
                retainDays: {type: integer, minimum: 1, default: 7}
```

2. Confirm the API server accepted it — the CRD's own status tells you:

```sh
kubectl get crd backupschedules.kubelings.dev -o jsonpath='{.status.conditions}' | jq
kubectl api-resources | grep backupschedule
kubectl explain backupschedule.spec        # your schema, in the built-in docs
```

3. Now the team's CR (top of the page) applies cleanly — do it, then poke the
   new API like any other:

```sh
kubectl -n kubelings get bks                       # shortName works
kubectl -n kubelings get backupschedule nightly-etcd -o yaml
```

<details>
<summary>Hint</summary>

If the CRD applies but `Established` is `False`, the names don't agree —
`metadata.name` must be exactly `plural.group`
(`backupschedules.kubelings.dev`). If the *CR* is rejected instead, that's
your own schema doing its job: `target: etcd` not `Etcd` (enum), `retainDays`
≥ 1. Validation you declare is validation the API server enforces — same
admission machinery as lesson 6.4.

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


## What you actually created

```
kubectl get bks
   │  discovery: api-resources said backupschedules exist (the CRD registered it)
   ▼
GET /apis/kubelings.dev/v1/namespaces/kubelings/backupschedules
   │  authn → RBAC (can this identity "get backupschedules"?) → admission
   ▼
etcd key: /registry/kubelings.dev/backupschedules/kubelings/nightly-etcd
```

Every mechanism from this module applies unchanged: stored in etcd (7.3),
watchable (7.1), RBAC-governed with `apiGroups: ["kubelings.dev"]` (6.1),
schema-validated at admission. Custom resources are not second-class — most
of "modern Kubernetes" (cert-manager's `Certificate`, Prometheus's
`ServiceMonitor`, Argo's `Application`, Gateway API's `HTTPRoute`) is exactly
this feature.

## CRD + controller = operator

Right now `nightly-etcd` is **inert data** — a wish written down. Nothing
watches it; no backup will run at 2 a.m. That's the honest half-truth of this
lesson, and it's the definition that matters:

```
CRD        = the noun        (API + storage + validation)   ← you did this
controller = the verb        (watch CRs, reconcile reality)  ← lesson 7.1's loop
operator   = noun + verb     (packaged operational knowledge)
```

An operator for this CRD would run the reconcile loop from 7.1 against
`BackupSchedule` objects: observe spec → create CronJobs (2.5) → write
`.status` (last backup time, success) → repeat. "Operator" is not magic
machinery — it's *someone encoded the runbook as a controller*.

## Field notes for real CRD work

- **`kubectl explain` works on CRs** *because* you wrote the openAPIV3Schema.
  A CRD without a schema is a YAML junk drawer — always write one; `enum`,
  `required`, and `default` catch at admission what would otherwise be a
  3 a.m. controller crash.
- **Versions are a contract**: `served` (does the API answer) vs `storage`
  (what's written to etcd — exactly one). v1→v2 migrations are Reddit's
  Pi-Day lesson (9.3) in miniature: deprecations are announced in
  `.spec.versions`, and clients that never re-read them break years later.
- **Deleting a CRD deletes every CR of that kind, everywhere.** It's the
  namespace-finalizer trap (3.5) at cluster scope — treat `kubectl delete
  crd` as an outage-grade command.
- **Watch out for webhook-converted CRDs** (conversion webhooks): they add
  the 6.3 failure mode — webhook down, CRs unreadable.

</details>
