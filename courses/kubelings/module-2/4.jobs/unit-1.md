---
kind: unit
title: "The Job That Never Finishes"
name: jobs-unit
---


## The situation

A nightly **`data-import`** Job in the `kubelings` namespace never finishes. The
downstream step waits for `kubectl wait --for=condition=complete job/data-import`
and hangs forever. The pod looks "Running" indefinitely.

A Job is **complete only when its container exits 0**. This one runs
`sleep infinity` — it never exits, so the Job stays active forever.

## Your task

Make `data-import` **run to completion** (a Job named `data-import` in `kubelings`
with `status.succeeded ≥ 1` and condition `Complete=True`).

```sh
kubectl -n kubelings get job data-import -o yaml | less
kubectl -n kubelings logs job/data-import
```

> Job pod templates are immutable — to change the command you must delete and
> recreate the Job.

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings delete job data-import
# recreate with a command that does the work and EXITS 0, e.g.:
#   command: ["sh","-c","echo importing...; sleep 2; echo done"]
```

See `solution.md`.

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

The Job's container ran `sleep infinity`. A Job only reports `Complete` when its
pod's container **exits 0** — a process that never exits keeps the Job `Active`
forever.

## Fix

Job pod templates are immutable, so delete and recreate with a command that
finishes:

```sh
kubectl -n kubelings delete job data-import

kubectl -n kubelings apply -f - <<'EOF'
apiVersion: batch/v1
kind: Job
metadata:
  name: data-import
spec:
  backoffLimit: 4
  template:
    spec:
      restartPolicy: OnFailure
      containers:
        - name: importer
          image: ghcr.io/iximiuz/labs/busybox:latest
          command: ["sh","-c","echo importing...; sleep 2; echo done"]
EOF
```

## Verify

```sh
kubectl -n kubelings wait --for=condition=complete job/data-import --timeout=60s
kubectl -n kubelings get job data-import
```

</details>
