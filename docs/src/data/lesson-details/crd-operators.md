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
