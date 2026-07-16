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
