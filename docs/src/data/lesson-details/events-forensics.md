## The situation

`analytics-job` was deployed an hour ago and has never run:

```
NAME                             READY   STATUS    RESTARTS   AGE
analytics-job-6c8d7b9f5d-x2kwp   0/1     Pending   0          1h
```

`kubectl logs` gives you nothing — the container never started, there's no
process to have logged. This is where a lot of engineers get stuck: **no logs
feels like no information.** But the cluster has been narrating the whole time,
in a stream most people ignore until they're desperate:

```sh
kubectl -n kubelings get events --sort-by=.lastTimestamp | tail -20
```

```
... Warning  FailedScheduling  pod/analytics-job-...  0/3 nodes are available:
     3 Insufficient memory.
```

There's your cause, timestamped and specific. **Events** are short-lived
(~1 hour TTL) records that every component emits about objects: scheduling
decisions, image pulls, probe failures, OOM kills, evictions, volume mounts.
They're the timeline of *what the cluster did and why*.

## Your task

1. Read the events for `analytics-job` and find why it can't schedule.
2. Fix the root cause (right-size the request to something a node can actually
   provide — this cluster's nodes are small).
3. It goes Available.

```sh
kubectl -n kubelings describe pod -l app=analytics-job | grep -A6 Events
kubectl get nodes -o custom-columns=NAME:.metadata.name,MEM:.status.allocatable.memory
```
