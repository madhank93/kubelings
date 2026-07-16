## The situation

The `cache` Deployment in `kubelings` never stays up — pods start, then die with
**OOMKilled** and fall into CrashLoopBackOff. The container needs roughly 50Mi of
working memory, but its `limits.memory` is set to **20Mi**, so the kernel kills it
the moment it allocates.

## Your task

Right-size the memory so `cache` runs steadily:

1. Inspect why the pod is being killed.
2. Raise the memory `requests`/`limits` to fit the ~50Mi workload (give headroom).
3. The Deployment must become Available and stop restarting.

```sh
kubectl -n kubelings get pods -l app=cache
kubectl -n kubelings describe pod -l app=cache | grep -A3 -i 'last state\|reason'
```
