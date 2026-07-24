---
kind: unit
title: "Expose a Deployment: your first Service"
name: expose-web-unit
---


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

<details>
<summary>Hint</summary>

The pods carry the label `app=web`. One command does everything:

```sh
kubectl -n kubelings expose deploy/web --port=80
```

Then watch the wiring appear:

```sh
kubectl -n kubelings get endpoints web   # two pod IPs = selector matched
```

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

<details>
<summary>Solution</summary>


## What a Service actually is

Three cooperating pieces:

1. **The Service object** — declares "port 80, pods matching `app=web`".
2. **The endpoints** — the control plane continuously lists pod IPs that match
   the selector *and* are Ready. This is the live routing table.
3. **kube-proxy + cluster DNS** — every node gets rules mapping the Service's
   virtual IP to those pod IPs; CoreDNS gives it the name `web.kubelings.svc`.

Change the pods (scale, reschedule, rolling update) and the endpoints follow.
Clients never notice.

## Fix

```sh
kubectl -n kubelings expose deploy/web --port=80
```

or declaratively:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: web
  namespace: kubelings
spec:
  selector:
    app: web
  ports:
    - port: 80
      targetPort: 80
```

## Verify

```sh
kubectl -n kubelings get endpoints web
# NAME   ENDPOINTS                     → two pod IPs, port 80
kubectl -n kubelings run tmp --rm -it --restart=Never --image=busybox:1.36 \
  -- wget -qO- http://web.kubelings.svc/
```

## Prevention / habits

- **Endpoints are the truth.** A Service with empty endpoints routes nowhere —
  it's the first thing to check when "the service is down" (next lesson!).
- Ship the Service in the same manifest/chart as the Deployment so a workload is
  never deployed unreachable.

</details>
