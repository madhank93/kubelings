---
kind: unit
title: "Autoscale a Deployment with an HPA (1 → 5)"
name: hpa-unit
---


## The situation

The `php-apache` Deployment in `kubelings` runs a single replica and falls over
under load. metrics-server is already collecting CPU usage. You need a
**HorizontalPodAutoscaler** so the app scales out on CPU and back in when idle.

## Your task

Create an HPA named **`php-apache`** that:

1. Targets the `php-apache` Deployment.
2. `minReplicas: 1`, `maxReplicas: 5`.
3. Scales on CPU (e.g. target 50% average utilization).

```sh
kubectl -n kubelings top pods          # metrics-server is live
kubectl -n kubelings get hpa
```

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings autoscale deploy php-apache --cpu=50% --min=1 --max=5
```

(Generate a load and watch with `kubectl -n kubelings get hpa -w` to see it climb.)

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
