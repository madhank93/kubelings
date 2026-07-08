---
kind: unit
title: "Events forensics: reconstruct the crime"
name: events-forensics-unit
---


## The situation

`analytics-job` was deployed an hour ago and has never run:

```
NAME                             READY   STATUS    RESTARTS   AGE
analytics-job-6c8d7b9f5d-x2kwp   0/1     Pending   0          1h
```

`kubectl logs` gives you nothing — the container never started, there's no
process to have logged. This is where a lot of engineers get stuck: **no logs
feels like no information.** But the cluster has been narrating the whole time,
in a stream most people ignore until they're desperate:

```sh
kubectl -n kubelings get events --sort-by=.lastTimestamp | tail -20
```

```
... Warning  FailedScheduling  pod/analytics-job-...  0/3 nodes are available:
     3 Insufficient memory.
```

There's your cause, timestamped and specific. **Events** are short-lived
(~1 hour TTL) records that every component emits about objects: scheduling
decisions, image pulls, probe failures, OOM kills, evictions, volume mounts.
They're the timeline of *what the cluster did and why*.

## Your task

1. Read the events for `analytics-job` and find why it can't schedule.
2. Fix the root cause (right-size the request to something a node can actually
   provide — this cluster's nodes are small).
3. It goes Available.

```sh
kubectl -n kubelings describe pod -l app=analytics-job | grep -A6 Events
kubectl get nodes -o custom-columns=NAME:.metadata.name,MEM:.status.allocatable.memory
```

<details>
<summary>Hint</summary>

`900Gi` requested; nodes have a few GB. Right-size:

```sh
kubectl -n kubelings set resources deploy/analytics-job \
  --requests=memory=64Mi --limits=memory=128Mi
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


## Reading events like a detective

```sh
# timeline across the whole namespace (default sort is NOT chronological — fix it):
kubectl -n kubelings get events --sort-by=.lastTimestamp

# only the object you care about:
kubectl -n kubelings describe pod <name>        # Events: section at the bottom
kubectl -n kubelings get events --field-selector involvedObject.name=<pod>

# only the bad news:
kubectl -n kubelings get events --field-selector type=Warning
```

The event told you exactly: `0/3 nodes ... Insufficient memory`. That's the
scheduler's **filter phase** (Module 7.2) reporting which predicate every node
failed — the same census format as the taints and access-mode lessons. One
vocabulary, everywhere.

## The catch that bites everyone: events expire

Default retention is ~1 hour. The 3 a.m. incident whose events you need at 9 a.m.
are **gone**. Consequences:

- For a *current* problem, events are gold — check them first.
- For *post-mortems*, you need events shipped somewhere durable (an event
  exporter → your logging stack). Ephemeral by default is a deliberate trap.

## The triage decision tree you now own

| Pod status | First command |
|---|---|
| `Pending` | **events** — scheduling/quota (this lesson) |
| `ContainerCreating` (stuck) | **events** — volume/secret mount (Module 3) |
| `ImagePullBackOff` | **events** — registry/tag (Module 1) |
| `CrashLoopBackOff` | **logs --previous** — the app spoke (Module 1) |
| `OOMKilled` | events + `describe` last state (Module 2) |
| `Running` but wrong | **logs**, then exec |

Events for anything that failed *before the process ran*; logs for anything that
ran and died. That split is most of day-one on-call.

## Prevention

- `kubectl get events -A --field-selector type=Warning` as a routine cluster
  health glance — Warnings are the cluster raising its hand.
- Ship events to durable storage if you ever want to debug yesterday.
- Alert on `FailedScheduling` older than a few minutes — it's capacity, quota,
  or an impossible request, and it never fixes itself.

</details>
