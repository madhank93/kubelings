## The situation

It's your first on-call shift. The team just shipped a new `orders` service to the
`kubelings` namespace and Slack is already unhappy: *"orders is down, can you look?"*

You check — and it's the classic sight:

```
NAME                      READY   STATUS             RESTARTS      AGE
orders-6f7d9c5b8d-4x2lp   0/1     CrashLoopBackOff   4 (12s ago)   2m
orders-6f7d9c5b8d-tk9wz   0/1     CrashLoopBackOff   4 (18s ago)   2m
```

The image builds fine. It ran on the developer's laptop. Kubernetes even says the
pod *starts* — it just dies within a second, every time, and the backoff delay
between attempts keeps doubling.

`CrashLoopBackOff` is not an error. It's Kubernetes telling you: *"the container
keeps exiting, and I'm pausing between retries."* The actual error is somewhere
else — and there is exactly one reliable place to find it.

## Your task

Make `orders` run steadily (2/2 Available, no CrashLoopBackOff):

1. Look at *why* the container exits — its exit code and last state.
2. Read what the process itself said on the way down.
3. Fix the Deployment accordingly. Don't touch the image — it's fine.

```sh
kubectl -n kubelings get pods -l app=orders
kubectl -n kubelings describe pod -l app=orders | grep -A5 -i 'last state'
kubectl -n kubelings logs -l app=orders --previous --tail=20
```
