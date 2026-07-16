## The situation

Friday, 16:58. A one-line hotfix ships. The deploy pipeline goes green — it only
checks that `kubectl apply` succeeded — and everyone goes home.

Monday:

```
NAME                        READY   STATUS             RESTARTS   AGE
frontend-7c9f6d4b8b-2mhx4   0/1     ImagePullBackOff   0          2d
frontend-7c9f6d4b8b-p8wvz   0/1     ImagePullBackOff   0          2d
```

`ImagePullBackOff` is the kubelet saying: *"I asked the registry for this image
and it said no — I'll retry with backoff."* The pod never even started; there is
no process, no logs. Which is the tell: **crash loops have logs, pull failures
have events.**

```sh
kubectl -n kubelings describe pod -l app=frontend | grep -A4 -i events
```

```
Failed to pull image "nginx:1.27.9999-alpine":
  ... nginx:1.27.9999-alpine: not found
```

The registry is up. The `nginx` repo exists. The *tag* `1.27.9999-alpine` was a
fat-fingered version that never existed — and `kubectl apply` will happily ship
a reference to nothing.

## Your task

Get `frontend` Available (2/2):

1. Confirm from events *why* the pull fails (don't guess — read it).
2. Point the Deployment at a tag that exists (`1.27-alpine` is fine).
3. Watch the rollout replace the stuck pods.

```sh
kubectl -n kubelings get pods -l app=frontend
kubectl -n kubelings describe pod -l app=frontend | tail -12
```
