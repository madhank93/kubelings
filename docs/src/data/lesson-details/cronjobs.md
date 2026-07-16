## The situation

The `report` CronJob in `kubelings` is scheduled **every minute**, but each run
takes ~90 seconds. With `concurrencyPolicy: Allow`, Kubernetes starts a fresh run
on schedule even though the previous one is still running — so active Jobs pile up
and pods accumulate until the namespace runs out of resources.

## Your task

Configure `report` so a new run **does not start while the previous run is still
active** (set `concurrencyPolicy` to `Forbid`, or `Replace` if you'd rather kill
the old run and start fresh).

```sh
kubectl -n kubelings get cronjob report -o yaml | grep concurrencyPolicy
kubectl -n kubelings get jobs
```
