---
kind: challenge

title: "The Job That Never Finishes"
description: |
  A one-shot data-import Job runs forever and never reports completion, so the
  pipeline that waits on it is stuck. Diagnose why, then make the Job actually
  complete.

categories:
- kubernetes

tagz:
- ckad
- workloads
- jobs

difficulty: easy

createdAt: 2026-06-30

playground:
  name: k8s-omni

tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 120
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      # Broken Job: the container never exits, so the Job never completes.
      kubectl apply -n "$NS" -f - <<'YAML'
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
                # BUG: sleeps forever instead of doing the import and exiting 0.
                command: ["sh","-c","echo importing...; sleep infinity"]
      YAML

  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      kubectl -n "$NS" get job data-import >/dev/null 2>&1 || {
        echo "not yet: no Job 'data-import' in $NS"; exit 1; }
      complete=$(kubectl -n "$NS" get job data-import \
        -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' 2>/dev/null)
      succeeded=$(kubectl -n "$NS" get job data-import -o jsonpath='{.status.succeeded}' 2>/dev/null)
      if [ "$complete" = "True" ] && [ "${succeeded:-0}" -ge 1 ]; then
        echo "PASS — Job data-import completed (succeeded=$succeeded)."
        exit 0
      fi
      echo "not yet: Job data-import has not completed (Complete=$complete, succeeded=${succeeded:-0})."
      echo "A Job completes only when its pod's container exits 0."
      exit 1
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
