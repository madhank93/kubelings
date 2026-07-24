---
kind: unit
title: "VPA: the recommender that watched the wrong app"
name: vpa-unit
---


## The situation

M2.7 (`oomkill`) taught right-sizing the hard way: one workload, by hand.
The **Vertical Pod Autoscaler** industrializes it — where HPA (M2.6)
changes *how many* pods, VPA changes *how big* each pod's requests are,
based on observed usage. Three components, of which this cluster runs one:

- **recommender** — watches metrics, computes target requests (installed)
- updater — evicts pods whose requests drift too far (not installed)
- admission webhook — rewrites requests on pod creation (not installed)

Recommender-only + `updateMode: "Off"` is the *audit mode* real teams
start with: recommendations land in status, nothing touches your pods.

`billing-api` requests `cpu: 500m` — someone's guess. The VPA has been
"watching" for days:

```sh
kubectl -n kubelings get vpa billing-api
# NAME          MODE   CPU   MEM   PROVIDED   AGE
# billing-api   Off                           …      ← no CPU, no MEM, nothing
kubectl -n kubelings get vpa billing-api -o jsonpath='{.status}'
# {} — not even a condition
```

Empty status. Now read what it's pointed at:

```sh
kubectl -n kubelings get vpa billing-api -o jsonpath='{.spec.targetRef}'
# {"apiVersion":"apps/v1","kind":"Deployment","name":"billing"}
kubectl -n kubelings get deploy
# NAME          READY …
# billing-api   1/1        ← there is no "billing"
```

The service was renamed; the VPA wasn't. A `targetRef` to a nonexistent
workload doesn't error — the recommender simply has nothing to watch and
says nothing, forever. Same silent-observability failure as the typo'd
metric in M8.9: wrong reference, empty result, no alarm.

## Your task

1. Point the VPA at the real Deployment:

   ```sh
   kubectl -n kubelings patch vpa billing-api --type=merge \
     -p '{"spec":{"targetRef":{"name":"billing-api"}}}'
   ```

2. Wait a recommender cycle (~1 min), then read the verdict:

   ```sh
   kubectl -n kubelings get vpa billing-api \
     -o jsonpath='{.status.recommendation.containerRecommendations[0]}' | python3 -m json.tool
   ```

   You'll get `lowerBound` / `target` / `upperBound` — compare `target`
   against the template's 500m guess.

<details>
<summary>Hint</summary>

Recommendation still empty after a couple minutes? Check the pipeline
bottom-up, M8-style: `kubectl top pods -n kubelings` (metrics-server
serving?) → `kubectl -n kube-system logs deploy/vpa-recommender --tail=20`
(target found?). The recommender needs a few scrapes of history before its
first estimate.

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
kubectl -n kubelings patch vpa billing-api --type=merge \
  -p '{"spec":{"targetRef":{"name":"billing-api"}}}'
```

## Reading a recommendation

```json
{
  "containerName": "billing-api",
  "lowerBound":  {"cpu": "25m",  "memory": "…"},
  "target":      {"cpu": "~50m", "memory": "…"},
  "upperBound":  {"cpu": "…",    "memory": "…"}
}
```

- **target** — what the recommender would set requests to (the spinner
  burns far less than the 500m guess; that gap × replicas × services is
  the money VPA exists to find).
- **lowerBound / upperBound** — the eviction band: with the *updater*
  installed, pods whose requests fall outside it get evicted and resized.
  Bounds tighten as history accumulates.

## The modes, and why "Off" first

| updateMode | Behavior |
|---|---|
| `Off` | recommendations only — audit mode, zero risk |
| `Initial` | apply at pod creation, never evict |
| `Auto`/`Recreate` | updater evicts to resize — mind your PDBs (M2.11!) |

Weeks of `Off` recommendations are the safe on-ramp; flipping to `Auto`
without reading them first hands eviction rights to an estimator you
never audited. And the classic constraint: **don't point VPA (Auto) and
HPA at the same metric on the same workload** — one resizes, the CPU
*percentage* shifts, the other scales; they chase each other (M2.17's
flapping, with two controllers).

## Prevention / takeaway

- `targetRef` is an unvalidated reference — renames orphan it silently.
  Alert on VPAs with empty `.status.recommendation` older than an hour;
  that's the `absent()`-style guard from M8.9 applied here.
- Even without ever enabling Auto, VPA in Off mode is a free fleet-wide
  right-sizing report: the M2.14 throttling incident and the M2.7 OOMKill
  are both "requests were a guess" stories this report would have caught.
- Bounds sanity: `minAllowed`/`maxAllowed` on the VPA cap what it may
  recommend — set them from your node shapes (a target bigger than your
  largest node is M2.19's unschedulable-surge bug, autoscaler edition).

</details>
