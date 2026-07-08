---
kind: unit
title: "The Service that routes to nothing"
name: selector-mismatch-unit
---


## The situation

This one has everyone stumped, because *everything looks fine*.

The `api` pods: Running, 2/2, zero restarts. The `api` Service: exists, correct
port. DNS: `api.kubelings.svc` resolves instantly. And yet every single request:

```
wget: can't connect to remote host: Connection timed out
```

A teammate already "checked everything" and is now blaming the CNI plugin. It is
not the CNI plugin. It's almost never the CNI plugin.

There's one command nobody ran:

```sh
kubectl -n kubelings get endpoints api
```

```
NAME   ENDPOINTS   AGE
api    <none>      3m
```

`<none>`. The Service is a perfectly healthy signpost pointing at **zero pods**.
DNS resolves the Service's virtual IP fine — but behind that IP, the routing
table is empty, so connections go nowhere and time out.

## Your task

Get traffic flowing to `api`:

1. Compare what the Service *selects* with what the pods *are labeled*.
2. Fix the mismatch (fix the Service **or** the pod labels — your call).
3. Endpoints must show both pod IPs, and an in-cluster request must succeed.

```sh
kubectl -n kubelings get svc api -o jsonpath='{.spec.selector}'
kubectl -n kubelings get pods -l app=api --show-labels
kubectl -n kubelings get endpoints api
```

<details>
<summary>Hint</summary>

The Service selects `app=api-server`. The pods are labeled `app=api`. One
character-level typo, zero endpoints, total outage. Patch the Service:

```sh
kubectl -n kubelings patch svc api -p '{"spec":{"selector":{"app":"api"}}}'
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


## Root cause

The Service's selector said `app: api-server`; the pods carry `app: api`. Label
selection is exact string matching — near enough is not enough. With no matching
Ready pods, the endpoints list is empty, and a Service with empty endpoints
accepts DNS lookups but can deliver traffic to nothing.

This is one of the most common outages in real clusters, precisely because every
individual piece reports healthy. Nothing is broken — the pieces just aren't
*connected*.

## Fix

```sh
kubectl -n kubelings patch svc api -p '{"spec":{"selector":{"app":"api"}}}'
```

(Equally valid: relabel the pods via the Deployment template — but changing the
Service is one object, no rollout.)

## Verify

```sh
kubectl -n kubelings get endpoints api        # two pod IPs appear immediately
kubectl -n kubelings run tmp --rm -it --restart=Never --image=busybox:1.36 \
  -- wget -qO- http://api.kubelings.svc/
```

Endpoints update within seconds of the selector change — no restart needed. The
endpoints controller reacts to the edit, kube-proxy reprograms, done.

## Prevention

- **First responder rule:** "service is down but pods are fine" → check
  `kubectl get endpoints` before anything else. Empty endpoints = selector
  mismatch or pods not Ready, 95% of the time.
- Keep selector and labels in one chart/kustomization so they can't drift.
- `kubectl get pods -l <the-service's-selector>` — if it returns nothing, you've
  found the outage.

</details>
