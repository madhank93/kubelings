## The situation

A well-meaning PR titled *"add health checks ✅"* merged this morning. Since
then, `payments` has restarted 47 times:

```
NAME                        READY   STATUS    RESTARTS        AGE
payments-58bd7f6d9c-8k2lp   0/1     Running   9 (33s ago)     8m
payments-58bd7f6d9c-wq4jx   1/1     Running   8 (2m ago)      8m
```

The app is *fine*. Requests that reach it succeed. But every few seconds the
kubelet kills a container in cold blood — because the **liveness probe** told it
to.

Two probes, two very different promises:

- **Readiness** failing says: *"don't send me traffic right now"* → pod is
  removed from Service endpoints. Recoverable, gentle, no restart.
- **Liveness** failing says: *"I am wedged beyond recovery — kill me"* → kubelet
  **restarts the container**. Violent, and correct only when the app truly can't
  recover on its own.

A liveness probe that's wrong — wrong port, wrong path, too-tight timeout — is
an automated pod assassin with cluster-admin over your container's life.

## Your task

Stop the killings. `payments` must reach 2/2 Available with restarts stopped —
and it must still *have* a liveness probe (fix it, don't delete it):

```sh
kubectl -n kubelings describe pod -l app=payments | grep -B2 -A6 -i liveness
kubectl -n kubelings get events -n kubelings --sort-by=.lastTimestamp | grep -i liveness | tail -5
```
