---
kind: unit
title: "Incident replay — restartPolicy: Never did not mean never (Universe)"
name: incident-job-restartpolicy-unit
---


## The real incident

**Universe** (ticketing platform), 21 July 2017. Search and reporting went down —
and so did the cluster UI, because the cluster itself went unhealthy. The trigger
was a **fix**: a reindex Job, shipped to repair broken search indexing.

Source: [Universe — Search and Reporting Outage](http://status.universe.com/incidents/115n3vxqwzcf)

Their postmortem names the misunderstanding precisely: the Job carried
`restartPolicy: Never`, and the author believed that meant *this runs once and
stops*. It doesn't. **`restartPolicy` is the pod's contract with its kubelet, not
the Job's contract with the cluster.** `Never` only says: when this container
exits, don't restart it *in place*. The Job controller then notices the pod
failed and does its own thing — it creates **another pod**. The reindexer was
resource-hungry; every retry landed somewhere new, ate what it could, and died.
Nodes fell over, the failure cascaded, and the cluster stayed sick until a human
deleted the Job.

Three fields would have made this a boring afternoon. None of them were set.

## This cluster, right now

The same reindex Job is running in `kubelings` — same intent, same defaults:

```sh
kubectl -n kubelings get job search-reindex
kubectl -n kubelings get pods -l job-name=search-reindex   # the graveyard grows
kubectl -n kubelings logs -l job-name=search-reindex --tail=3
```

The container dies on a corrupt document. It will keep dying — that part is a
genuine bug in the reindexer, and **not your job today**. Your job is that this
bug currently has no blast-radius limit at all:

```sh
kubectl -n kubelings get job search-reindex -o jsonpath='{.spec.backoffLimit}{"\n"}'
# 2147483647  ← "forever", spelled in int32
```

<!-- d2:retry -->
```text
       ┌────────────┐      
       │pod exits 1 │      
       │            │      
       └────────────┘      
             │             
restartPolicy is the POD's 
             │             
             ▼             
   ┌──────────────────────┐
   │Job creates a new pod │
   │                      │
   └──────────────────────┘
             │             
  unbounded backoffLimit   
             │             
             ▼             
     ┌────────────────┐    
     │nodes exhausted │    
     │                │    
     └────────────────┘    
```
<!-- /d2:retry -->

## Your task

Make the failure bounded. Keep the reindexer failing — bound what its failure can
do:

1. `backoffLimit` **≤ 4** — the Job gives up instead of grinding.
2. `activeDeadlineSeconds` **≤ 300** — wall-clock has a ceiling too, for the
   failure mode where each attempt hangs rather than exits.
3. `ttlSecondsAfterFinished` set — finished Jobs and their pods collect
   themselves instead of accumulating.
4. `resources.limits` (cpu **and** memory) on the container — a retry storm must
   never be able to take a node with it.
5. The Job must end **Failed** on its own, and no more than `backoffLimit + 1`
   pods may remain.

Job specs are largely **immutable** — you'll delete and recreate, not patch.

<details>
<summary>Hint</summary>

Keep the command exactly as it is (it must still exit 1), and add the four
guardrails:

```sh
kubectl -n kubelings delete job search-reindex     # takes its pods with it
kubectl -n kubelings apply -f - <<'EOF'
apiVersion: batch/v1
kind: Job
metadata:
  name: search-reindex
  labels: {app: search}
spec:
  backoffLimit: 2
  activeDeadlineSeconds: 120
  ttlSecondsAfterFinished: 300
  template:
    metadata:
      labels: {app: search, role: reindex}
    spec:
      restartPolicy: Never
      containers:
        - name: reindex
          image: busybox:1.36
          command: ["sh","-c",'echo "reindexing search corpus…"; sleep 3; echo "FATAL: corrupt document 41c9"; exit 1']
          resources:
            requests: {cpu: 100m, memory: 32Mi}
            limits:   {cpu: 500m, memory: 128Mi}
EOF
kubectl -n kubelings get job search-reindex -w   # give it ~40s to give up
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

## Root cause: two restart systems, one field name

| Field | Owner | Means |
|---|---|---|
| `spec.template.spec.restartPolicy` | kubelet | restart the **container** in place? (`Never` / `OnFailure`) |
| `spec.backoffLimit` | Job controller | how many **pods** may fail before the Job gives up |
| `spec.activeDeadlineSeconds` | Job controller | how long the whole Job may run, ever |

`restartPolicy: Never` + default-ish `backoffLimit` is *not* "run once". With
`Never` the controller replaces the failed pod; with `OnFailure` the kubelet
restarts the container in the same pod (and `backoffLimit` still counts pod
failures). Neither is a bound unless you write the bound.

Universe's own remediation was the manual version of this: an engineer deleted
the Job, "after which all nodes autohealed fully healthy again."

## The fix, generalized

- **Every Job gets a `backoffLimit` and an `activeDeadlineSeconds`.** Retries are
  a budget, not a right. `activeDeadlineSeconds` catches the hang; `backoffLimit`
  catches the crash.
- **Every Job gets limits.** A crash-looping workload that also has no ceiling is
  how a bad Job becomes a bad *node* (M8's `incident-node-oom` is the same story
  from the memory side).
- **`ttlSecondsAfterFinished`** so completed/failed Jobs disappear — the pile-up
  is its own outage class (M2 `pattern-zombie-cronjobs`).
- **Alert on `Job.status.failed`**, not just on pods. A Job burning through
  retries at 3am looks like nothing on a pod dashboard.

## What retries actually cost

Job retries back off exponentially (10s, 20s, 40s… capped at 6 min), which sounds
gentle — but each attempt is a *fresh pod*: a scheduling decision, an image
check, resource requests granted again. A greedy container plus an unbounded
retry count is a slow-motion resource-exhaustion attack you wrote yourself.

## Prevention

```sh
# Jobs in your cluster with no wall-clock bound:
kubectl get jobs -A -o json |
  jq -r '.items[] | select(.spec.activeDeadlineSeconds == null) |
         "\(.metadata.namespace)/\(.metadata.name)"'
```

Put both bounds in the template every team copies from — the default that ships
inside your platform's Job scaffold is the one that ends up in production.

</details>
