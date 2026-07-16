## The situation

HPA (M2.6) scales on *resource pressure* — CPU, memory, metrics API. But
"we need 3 replicas during business hours" isn't pressure, and "scale on
queue depth / Kafka lag / cron windows" isn't in HPA's vocabulary. **KEDA**
extends it: 70+ event-source *scalers*, one CRD:

```
ScaledObject ──(KEDA operator)──▶ HPA (created & fed by KEDA)──▶ Deployment
```

KEDA doesn't replace HPA — it *manufactures* one and feeds it external
metrics through its own metrics apiserver. Everything M2.6 taught about
HPA behavior still applies underneath.

`report-web` should run 3 replicas through the working day. Reality:

```sh
kubectl -n kubelings get scaledobject
# NAME         …   READY   ACTIVE   TRIGGERS   AGE
# report-web   …   False   Unknown  cron       …
kubectl -n kubelings get deploy report-web    # 1/1 — forever
kubectl -n kubelings get hpa                  # nothing — KEDA won't build
                                              # an HPA from a broken spec
```

Ready=False. Ask why — same reflex as every controller in this course:

```sh
kubectl -n kubelings get scaledobject report-web \
  -o jsonpath='{.status.conditions[?(@.type=="Ready")].message}'
# error parsing cron schedule "0 25 * * *" …
kubectl -n keda logs deploy/keda-operator --tail=5    # same story, louder
```

The trigger's `start: 0 25 * * *` — minute 0 of hour **25**. The schedule
never parses, so the trigger can't even evaluate, so KEDA never creates
the HPA. Not "scales at the wrong time" — *structurally dead*, visible
only in conditions.

## Your task

1. Fix the window — business hours, say 08:00–23:59 UTC... but this drill
   must pass whenever you run it, so use the full day:

   ```sh
   kubectl -n kubelings patch scaledobject report-web --type=merge -p '
   {"spec":{"triggers":[{"type":"cron","metadata":{
     "timezone":"Etc/UTC",
     "start":"0 0 * * *",
     "end":"59 23 * * *",
     "desiredReplicas":"3"}}]}}'
   ```

2. Watch the machinery assemble itself:

   ```sh
   kubectl -n kubelings get scaledobject report-web    # READY True, ACTIVE True
   kubectl -n kubelings get hpa                        # keda-hpa-report-web appears
   kubectl -n kubelings get deploy report-web -w       # 1 → 3
   ```
