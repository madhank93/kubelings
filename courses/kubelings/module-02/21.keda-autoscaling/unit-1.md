---
kind: unit
title: "KEDA: the ScaledObject that never scaled"
name: keda-autoscaling-unit
---


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

<details>
<summary>Hint</summary>

The cron trigger holds `desiredReplicas` between `start` and `end`, and
returns to `minReplicaCount` outside. The one minute 23:59→00:00 is
outside the window above — if you're verifying exactly then, congrats on
finding the gap; wait sixty seconds. Real setups pair two windows or
accept the seam consciously.

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


## Fix

```sh
kubectl -n kubelings patch scaledobject report-web --type=merge -p '
{"spec":{"triggers":[{"type":"cron","metadata":{
  "timezone":"Etc/UTC",
  "start":"0 0 * * *",
  "end":"59 23 * * *",
  "desiredReplicas":"3"}}]}}'
```

## What KEDA actually did after the fix

```sh
kubectl -n kubelings get hpa keda-hpa-report-web -o yaml | grep -A4 metrics
# external metric: s0-cron-Etc-UTC-00xxx-5923xxx …
```

A real HPA, targeting an *external* metric served by KEDA's metrics
apiserver. The division of labor: scalers talk to event sources (cron
clocks, Kafka, SQS, Prometheus…), KEDA translates them into metrics, HPA
does what HPA does. Debugging therefore has two floors: ScaledObject
conditions (KEDA's floor) and the HPA's events (M2.6's floor).

Why the cron scaler here and not the fancy ones: it teaches the whole
ScaledObject lifecycle with zero external dependencies. A Prometheus or
queue trigger changes only the `metadata:` block — and drags a metrics
stack into what is structurally the same lesson.

## The 0-replica superpower (mentioned, deliberately not used)

`minReplicaCount: 0` is KEDA's headline feature — scale-to-zero, which
plain HPA cannot do (its floor is 1). KEDA handles the 0→1 wake-up itself,
then hands 1→N to the HPA. This drill keeps the floor at 1: waking from
zero adds cold-start behavior that deserves its own story.

## Prevention / takeaway

- **Ready=False on a ScaledObject means the HPA doesn't exist yet.** No
  HPA, no scaling, no error events on the Deployment — the failure lives
  entirely in the ScaledObject's conditions. Alert on ScaledObject
  Ready!=True the same way you alert on Deployment availability.
- Trigger `metadata:` is stringly-typed per scaler — nothing validates
  hour 25 at apply time (admission validation exists for some fields, not
  schedule semantics). CI-lint your schedules; the M6.12 pattern (test
  the policy by firing it) applies to autoscaling configs too.
- Don't stack a KEDA cpu trigger on a workload that also has its own HPA
  targeting CPU — two HPAs fighting over one scale subresource is M2.17
  flapping with paperwork.
- Windows have seams (23:59→00:00 here). Pair complementary windows or
  document the seam; "why did we briefly scale down at midnight" is a
  real page.

</details>
