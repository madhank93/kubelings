---
kind: unit
title: "Final boss: three faults, no hints"
name: final-boss-unit
---


## The situation

You're on call. A `checkout` stack shipped to `kubelings` and **nothing works
right**. Three services, three separate problems, layered — fixing one won't
turn the check green. There is no fault list and there are no hints. Just you,
`kubectl`, and everything you've learned.

```
api      Service exists, "can't reach it"
web      pods keep restarting
worker   pod won't start at all
```

This is the exam. Diagnose each from first principles, using the muscle memory
the last eight modules built.

## Your task

Bring all three to healthy:

- `api` — reachable at `http://api.kubelings.svc/` with 2 endpoints.
- `web` — 2/2 Available, restarts stopped.
- `worker` — scheduled and Available.

No hints in this unit. Your toolkit, in the order that solves things fastest:

```sh
kubectl -n kubelings get deploy,pods,svc,endpoints        # the wide view first
kubectl -n kubelings get pods                              # which symptom is which?
kubectl -n kubelings describe pod <name>                   # events (didn't-start faults)
kubectl -n kubelings logs <name> --previous                # logs (crashed faults)
kubectl -n kubelings get endpoints <svc>                   # the empty-endpoints tell
```

Map each symptom to its diagnostic:

| Symptom | Recall from |
|---|---|
| Service reachable-but-dead, pods healthy | endpoints / selectors — **M1** |
| Pods restarting with no app errors | probes — **M2** |
| Pod stuck `Pending` | events / scheduling / resources — **M5, M8** |

::simple-task
---
:tasks: tasks
:name: verify_done
---
#active
Solve all three — this check turns green only when every fault is fixed.

#completed
✅ Solved — you beat the final boss.
::

<details>
<summary>Solution (only after you've tried)</summary>


## Fault A — `api`: the Service that routes to nothing

```sh
kubectl -n kubelings get endpoints api        # <none>
kubectl -n kubelings get pods -l app=api --show-labels   # app=api
kubectl -n kubelings get svc api -o jsonpath='{.spec.selector}'   # app=api-server
```

Selector mismatch (Module 1). Fix:

```sh
kubectl -n kubelings patch svc api -p '{"spec":{"selector":{"app":"api"}}}'
```

## Fault B — `web`: the probe that kills healthy pods

```sh
kubectl -n kubelings describe pod -l app=web | grep -i 'liveness\|killing'
# Liveness probe failed: connection refused on :8080 ; app serves :80
```

Wrong liveness port (Module 2). Fix:

```sh
kubectl -n kubelings patch deploy web --type=json -p '[
  {"op":"replace","path":"/spec/template/spec/containers/0/livenessProbe/httpGet/port","value":80}
]'
```

## Fault C — `worker`: the impossible request

```sh
kubectl -n kubelings describe pod -l app=worker | grep -A3 Events
# 0/3 nodes available: Insufficient memory  (requested 500Gi)
```

Unschedulable request (Modules 5 & 8). Fix:

```sh
kubectl -n kubelings set resources deploy/worker \
  --requests=memory=32Mi --limits=memory=64Mi
```

## Why they had to be diagnosed separately

Each fault presents a *different* status and hides in a *different* place:

- **A** — nothing looks broken (pods Ready); the tell is `endpoints`.
- **B** — pods churn; the tell is *events/logs showing no app error* → probes.
- **C** — pod never starts; the tell is a `FailedScheduling` **event**.

The meta-skill isn't any one fix — it's **reading the symptom to pick the right
lens** (endpoints vs events vs logs) instead of guessing. That's the loop under
every lesson in this course, and under every entry in the
[Incident Library](https://kubelings.madhan.app/reference/incident-library/):
real outages are usually two or three of these, stacked, at 3 a.m.

## Where to go from here

You've run the platform from a single pod to a self-inflicted cascade. Next:

- Sweep the **Incident Library** — you can now read every one of those
  postmortems and see the mechanism, not just the story.
- Re-run any lesson with the solution hidden; speed is the last thing to build.
- The real final boss is production. You're ready to meet it with a diagnosis,
  not a guess.

</details>
