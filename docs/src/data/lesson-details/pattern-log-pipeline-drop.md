> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters (tail globs that match nothing, parsers that reject
> every line, rotation the agent never follows), not a specific company's
> incident.

## The situation

Someone opens an incident: *"we have no payment logs for last Tuesday."* You go
looking, and every dashboard you own says the pipeline is fine.

```sh
kubectl -n kubelings get pods -l app=payments        # 2/2 Running, no restarts
kubectl -n kubelings logs deploy/payments -c app --tail=3      # the app is logging
kubectl -n kubelings logs deploy/payments -c shipper --tail=20 # …the shipper isn't
```

The `payments` pod runs the vendor application plus a **fluent-bit sidecar** that
tails the app's log directory and forwards records. The app writes. The shipper
is Ready. Nothing arrives.

This is the log pipeline's defining property: **it fails open, quietly.** A tail
input whose glob matches no file is not an error — there is simply nothing to
tail, and the agent reports itself perfectly healthy while your audit trail goes
to `/dev/null`.

<figure class="lesson-diagram">
<img src="/diagrams/pattern-log-pipeline-drop-tail.svg" alt="pattern-log-pipeline-drop tail diagram" loading="lazy">
</figure>

## Your task

Get records flowing again — without touching the application:

1. The app keeps writing `/var/log/app/app.log`. That path belongs to the vendor;
   changing it is not the fix.
2. The shipper's `stdout` OUTPUT stays (that's the "backend" in this drill).
3. `kubectl -n kubelings logs deploy/payments -c shipper` must show the app's
   records — look for `order_id`.

```sh
kubectl -n kubelings exec deploy/payments -c app -- ls -l /var/log/app
kubectl -n kubelings get configmap shipper-config -o jsonpath='{.data.fluent-bit\.conf}'
```
