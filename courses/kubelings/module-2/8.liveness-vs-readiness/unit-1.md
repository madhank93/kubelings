---
kind: unit
title: "The liveness probe that kills healthy pods"
name: liveness-vs-readiness-unit
---


## The situation

A well-meaning PR titled *"add health checks ✅"* merged this morning. Since
then, `payments` has restarted 47 times:

```
NAME                        READY   STATUS    RESTARTS        AGE
payments-58bd7f6d9c-8k2lp   0/1     Running   9 (33s ago)     8m
payments-58bd7f6d9c-wq4jx   1/1     Running   8 (2m ago)      8m
```

The app is *fine*. Requests that reach it succeed. But every few seconds the
kubelet kills a container in cold blood — because the **liveness probe** told it
to.

Two probes, two very different promises:

- **Readiness** failing says: *"don't send me traffic right now"* → pod is
  removed from Service endpoints. Recoverable, gentle, no restart.
- **Liveness** failing says: *"I am wedged beyond recovery — kill me"* → kubelet
  **restarts the container**. Violent, and correct only when the app truly can't
  recover on its own.

A liveness probe that's wrong — wrong port, wrong path, too-tight timeout — is
an automated pod assassin with cluster-admin over your container's life.

## Your task

Stop the killings. `payments` must reach 2/2 Available with restarts stopped —
and it must still *have* a liveness probe (fix it, don't delete it):

```sh
kubectl -n kubelings describe pod -l app=payments | grep -B2 -A6 -i liveness
kubectl -n kubelings get events -n kubelings --sort-by=.lastTimestamp | grep -i liveness | tail -5
```

<details>
<summary>Hint</summary>

Events say it plainly: `Liveness probe failed: ... connect: connection refused`
on port **8080**. nginx serves on **80** — same port readiness already checks.

```sh
kubectl -n kubelings patch deploy payments --type=json -p '[
  {"op":"replace","path":"/spec/template/spec/containers/0/livenessProbe/httpGet/port","value":80}
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

The liveness probe targeted port `8080`; the container serves on `80`. Every
probe attempt got `connection refused`, and after `failureThreshold: 2` the
kubelet restarted a perfectly healthy container — forever.

Note the trap in the pod listing: readiness (correct port) showed the pod READY
between kills, so the service *mostly* worked. Mostly-working is how probe bugs
survive review.

## Fix

```sh
kubectl -n kubelings patch deploy payments --type=json -p '[
  {"op":"replace","path":"/spec/template/spec/containers/0/livenessProbe/httpGet/port","value":80}
]'
kubectl -n kubelings rollout status deploy/payments
```

## Choosing probes (the honest rules)

- **Readiness:** check the full serving path (can I actually answer requests?).
  Fail it during startup, overload, dependency loss.
- **Liveness:** check only *"is the process wedged?"* — the shallowest possible
  check (does the HTTP server accept?). Never include dependencies: if the
  database is down, restarting *you* won't fix the database, but it will turn a
  degradation into a restart storm.
- Restart storms during incidents are very often probe design, not app bugs.

## Prevention

- Liveness ≠ readiness copy-paste. Different questions, different checks.
- Generous `failureThreshold` × `periodSeconds` on liveness — you want restarts
  to be rare, deliberate, last-resort.
- Watch `kubectl get events | grep -i liveness` after every probe change.

</details>
