## The situation

You're on call. A `checkout` stack shipped to `kubelings` and **nothing works
right**. Three services, three separate problems, layered — fixing one won't
turn the check green. There is no fault list and there are no hints. Just you,
`kubectl`, and everything you've learned.

```
api      Service exists, "can't reach it"
web      pods keep restarting
worker   pod won't start at all
```

This is the exam. Diagnose each from first principles, using the muscle memory
the last eight modules built.

## Your task

Bring all three to healthy:

- `api` — reachable at `http://api.kubelings.svc/` with 2 endpoints.
- `web` — 2/2 Available, restarts stopped.
- `worker` — scheduled and Available.

No hints in this unit. Your toolkit, in the order that solves things fastest:

```sh
kubectl -n kubelings get deploy,pods,svc,endpoints        # the wide view first
kubectl -n kubelings get pods                              # which symptom is which?
kubectl -n kubelings describe pod <name>                   # events (didn't-start faults)
kubectl -n kubelings logs <name> --previous                # logs (crashed faults)
kubectl -n kubelings get endpoints <svc>                   # the empty-endpoints tell
```

Map each symptom to its diagnostic:

| Symptom | Recall from |
|---|---|
| Service reachable-but-dead, pods healthy | endpoints / selectors — **M1** |
| Pods restarting with no app errors | probes — **M2** |
| Pod stuck `Pending` | events / scheduling / resources — **M5, M8** |
