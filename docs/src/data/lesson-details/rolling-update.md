## The situation

The `web` Deployment in the `kubelings` namespace serves production traffic, but
every release causes a brief total outage. Its `RollingUpdate` strategy is set to
`maxSurge: 0` and `maxUnavailable: 100%` — so on each rollout Kubernetes is allowed
to terminate **all** pods before any replacement is Ready, and is **not** allowed
to surge a single extra pod to cover the gap.

## Your task

Make `web`'s rolling update **zero-downtime**:

1. Set `maxSurge` so at least one new pod can start before old ones go away.
2. Set `maxUnavailable` so the whole fleet can't be taken down at once.
3. Keep the Deployment Available (all 3 replicas Ready).

```sh
kubectl -n kubelings get deploy web -o yaml | less
```
