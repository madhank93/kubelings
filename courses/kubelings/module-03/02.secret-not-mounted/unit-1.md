---
kind: unit
title: "ContainerCreating forever: the Secret that isn't"
name: secret-not-mounted-unit
---


## The situation

The new API gateway should have been serving ten minutes ago:

```
NAME                       READY   STATUS              RESTARTS   AGE
gateway-5f8d9b7c6d-mq2xk   0/1     ContainerCreating   0          10m
```

`ContainerCreating` for a few seconds is normal (image pull, volume setup). Ten
*minutes* means a volume can't be assembled. Events tell you which one:

```sh
kubectl -n kubelings describe pod -l app=gateway | tail -6
```

```
MountVolume.SetUp failed for volume "tls" :
  secret "gateway-tls" not found
```

There *is* a TLS secret in the namespace — the cert-rotation job faithfully
maintains `gateway-tls-cert`. The Deployment asks for `gateway-tls`. Two teams,
two naming conventions, zero communication.

Note the design: Kubernetes **refuses to start the container at all** rather
than start it without its Secret. A gateway that boots without TLS material and
serves plaintext would be a much worse failure than a pod that visibly never
starts. The hang is the safety feature.

## Your task

Get `gateway` Running with the cert actually mounted at `/etc/tls/`:

```sh
kubectl -n kubelings get secrets
kubectl -n kubelings get deploy gateway -o jsonpath='{.spec.template.spec.volumes}'
```

Fix from whichever side is right — but the rotation job *rewrites*
`gateway-tls-cert` on every rotation, so renaming the Secret means fighting a
robot forever. Fix the reference.

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch deploy gateway --type=json -p '[
  {"op":"replace","path":"/spec/template/spec/volumes/0/secret/secretName","value":"gateway-tls-cert"}
]'
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


## Root cause

The volume referenced Secret `gateway-tls`; the actual object is
`gateway-tls-cert`. The kubelet retried volume setup forever, holding the pod in
`ContainerCreating`.

## The three ways config references fail

| Failure | Status | Where it says so |
|---|---|---|
| Secret/ConfigMap **object** missing (volume) | `ContainerCreating`, forever | events: `MountVolume.SetUp failed` |
| **Key** missing inside existing object (env) | `CreateContainerConfigError` | events: `couldn't find key` |
| Object missing but `optional: true` | pod **starts** with empty/missing files | nowhere — silent (Module 2's init-container trap) |

Same typo, three different symptoms depending on how the config is wired. Learn
the mapping and `describe pod` becomes a one-look diagnosis.

## Fix

```sh
kubectl -n kubelings patch deploy gateway --type=json -p '[
  {"op":"replace","path":"/spec/template/spec/volumes/0/secret/secretName","value":"gateway-tls-cert"}
]'
kubectl -n kubelings rollout status deploy/gateway
```

## Prevention

- Secret names are an API contract between producer (rotation job) and consumer
  (deployment) — put the name in one shared values file, not two heads.
- Alert on pods in `ContainerCreating` > 2 minutes; it is always a volume or
  runtime problem, never "just slow."
- Bonus: mounted Secrets update in place on rotation (like ConfigMaps) — another
  reason the mount, not env, is the right wiring for certs.

</details>
