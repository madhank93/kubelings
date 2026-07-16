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
