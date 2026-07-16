## The situation

The frontend team shipped `web` — and it's healthy. Two nginx pods, Running,
zero restarts:

```
NAME                   READY   STATUS    RESTARTS   AGE
web-5b8c49d7c6-8fj2m   1/1     Running   0          1m
web-5b8c49d7c6-qz7xk   1/1     Running   0          1m
```

And yet the backend team says every call to `http://web/` fails:
*"could not resolve host: web"*.

Here's the thing pods don't give you: a **stable address**. Pod IPs change on
every reschedule, and nothing load-balances between the two replicas. That's the
Service's job — a fixed virtual IP and DNS name (`web.kubelings.svc`) that
continuously tracks whichever pods match its **label selector**.

No Service exists yet. The pods are healthy and completely unreachable.

## Your task

Create a Service named `web` in `kubelings` so in-cluster clients can reach the
frontend:

1. It must be named `web` and expose **port 80**.
2. Its selector must match the running pods (look at their labels).
3. Prove it: the check curls `http://web.kubelings.svc/` from inside the cluster.

```sh
kubectl -n kubelings get pods -l app=web --show-labels
kubectl -n kubelings get svc,endpoints
```
