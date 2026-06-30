# Solution — CronJob concurrencyPolicy

## Root cause

`report` runs every minute but each run lasts ~90s, and `concurrencyPolicy: Allow`
lets a new run start on schedule regardless of in-flight runs. Overlapping Jobs
accumulate until resources are exhausted.

## Fix

```sh
kubectl -n kubelings patch cronjob report --type=merge \
  -p '{"spec":{"concurrencyPolicy":"Forbid"}}'
```

- `Forbid` — skip the new run if the previous one is still active.
- `Replace` — cancel the running one and start the new one instead.

## Verify

```sh
kubectl -n kubelings get cronjob report \
  -o jsonpath='{.spec.concurrencyPolicy}{"\n"}'
kubectl -n kubelings get jobs   # runs no longer stack up
```
