---
kind: unit
title: "Slow starter vs impatient liveness"
name: startup-probe-unit
---


## The situation

`legacy-erp` is twenty years of business logic in one container. It works — but
it takes ~40 seconds to boot: cache warm-up, migrations, an in-memory index.

Someone gave it a standard liveness probe: first check at t=10s, two failures
allowed, 5s apart. Do the math:

```
t=10s  liveness check #1 — /tmp/healthy missing (still booting) → fail 1
t=15s  liveness check #2 — still booting → fail 2 → KILL
t=15s  container restarts, boot begins again from zero
t=25s  fail, fail → KILL
...forever
```

The pod is trapped in a time loop. It is **never unhealthy** — it's just slower
than the probe's patience. And each kill throws away the boot progress, so the
40-second app never gets 40 seconds.

This is exactly the gap `startupProbe` was invented for.

## Your task

`legacy-erp` must reach Available with its liveness protection intact:

1. Keep the livenessProbe (it's correct for a *booted* app).
2. Add a `startupProbe` that tolerates the slow boot — while it runs, liveness
   and readiness are suspended.
3. New pod must boot without being killed (restarts ≤ 1).

```sh
kubectl -n kubelings get pods -l app=legacy-erp
kubectl -n kubelings describe pod -l app=legacy-erp | grep -A4 -i 'liveness\|killing'
```

<details>
<summary>Hint</summary>

Same check, but as a startupProbe with a big failure budget:

```sh
kubectl -n kubelings patch deploy legacy-erp --type=strategic -p '
spec:
  template:
    spec:
      containers:
        - name: legacy-erp
          startupProbe:
            exec:
              command: ["test", "-f", "/tmp/healthy"]
            periodSeconds: 5
            failureThreshold: 18
'
```

`18 × 5s = 90s` of allowed boot time. Liveness doesn't fire until startup passes.

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

Liveness assumed "no answer in 20 seconds = dead", but this app's healthy boot
takes 40. Every restart reset boot progress, so the failure was self-sustaining
— the classic *impatient liveness* time loop.

## Why startupProbe, not a bigger initialDelaySeconds

You *could* set `initialDelaySeconds: 60` on liveness. Two problems:

1. It delays protection forever: a pod that wedges 5 seconds in sits undetected
   for the full 60s on **every** restart, not just first boot.
2. Boot time varies (cold cache vs warm node). A fixed delay is either wasteful
   or still too short on a bad day.

`startupProbe` separates the phases: a generous budget that applies **only while
starting** (liveness/readiness suspended), then the tight liveness takes over
the moment startup passes. Slow boot tolerated, wedged runtime still caught fast.

## Fix

```sh
kubectl -n kubelings patch deploy legacy-erp --type=strategic -p '
spec:
  template:
    spec:
      containers:
        - name: legacy-erp
          startupProbe:
            exec:
              command: ["test", "-f", "/tmp/healthy"]
            periodSeconds: 5
            failureThreshold: 18
'
kubectl -n kubelings rollout status deploy/legacy-erp   # takes ~40s — that's the point
```

## Prevention

- Any container with >10s boot: startupProbe, always.
- Budget = worst observed boot × 2.
- Watch for the signature in events: `Killing` + restart counts climbing with
  *no* app error logs = probes, not the app.

</details>
