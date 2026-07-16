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
