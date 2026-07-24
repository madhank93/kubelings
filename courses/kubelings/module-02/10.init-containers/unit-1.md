---
kind: unit
title: "Stuck at Init: the gate that never opens"
name: init-containers-unit
---


## The situation

```
NAME                       READY   STATUS     RESTARTS   AGE
reports-7f6b8d9c4d-x2kwm   0/1     Init:0/1   0          22m
```

`Init:0/1` — the pod hasn't even *started* being your application yet. Init
containers run **before** app containers, strictly in order, each to completion.
Until the last one exits 0, the app containers don't exist. That's the feature:
a programmable gate for "don't start until X is true."

This gate is waiting for a config file to appear in a mounted volume… and it has
been "waiting for config volume..." for 22 minutes. The ConfigMap it wants
*exists*. So why is the volume empty?

Look closely at the two names in the pod spec.

## Your task

Get `reports` Running and Available:

1. Read the init container's logs — init containers have logs too
   (`kubectl logs <pod> -c <init-container-name>`).
2. Compare the volume's ConfigMap reference against what actually exists.
3. Fix the reference.

```sh
kubectl -n kubelings get pods -l app=reports
kubectl -n kubelings logs -l app=reports -c wait-for-config --tail=5
kubectl -n kubelings get configmaps
kubectl -n kubelings get deploy reports -o jsonpath='{.spec.template.spec.volumes}'
```

<details>
<summary>Hint</summary>

The volume references ConfigMap `report-config` (marked `optional: true`, so the
volume mounts *empty* instead of erroring). The real ConfigMap is
`reports-config` — one letter. Fix the reference:

```sh
kubectl -n kubelings patch deploy reports --type=json -p '[
  {"op":"replace","path":"/spec/template/spec/volumes/0/configMap/name","value":"reports-config"}
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

Two bugs conspiring:

1. The volume referenced `report-config`; the ConfigMap is `reports-config`.
2. The reference was `optional: true` — so instead of a loud
   `CreateContainerConfigError`, Kubernetes silently mounted an **empty volume**,
   and the init container waited politely, forever, for a file that would never
   come.

`optional: true` converts a crash into a hang. Sometimes that's what you want.
Usually it isn't.

## Fix

```sh
kubectl -n kubelings patch deploy reports --type=json -p '[
  {"op":"replace","path":"/spec/template/spec/volumes/0/configMap/name","value":"reports-config"}
]'
kubectl -n kubelings rollout status deploy/reports
```

## Init container mechanics worth knowing

- Run **serially, in order**, each to completion, before any app container.
- A failing init container restarts per the pod's restartPolicy →
  `Init:CrashLoopBackOff` is a thing; check init logs with `-c <name>`.
- Status decoder: `Init:0/1` = first of one still running; `Init:Error` /
  `Init:CrashLoopBackOff` = it's failing, not waiting.
- Readiness/liveness probes don't apply to init containers — your only lens is
  logs and events.

## Prevention

- Log *what* you're waiting for and *where you're looking* inside wait loops —
  this init container's log said "waiting" but not the path/name it polled.
- Think twice before `optional: true` on config that isn't optional.
- Put a ceiling on waits (`timeout 300 sh -c 'until …'`) so a bad reference
  becomes a visible failure, not an eternal hang.

</details>
