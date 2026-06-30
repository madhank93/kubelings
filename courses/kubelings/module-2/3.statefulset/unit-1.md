---
kind: unit
title: "StatefulSet with Stable Pod Identity + Headless Service"
name: statefulset-unit
---


## The situation

You're deploying a clustered, stateful workload where peers must address each
other by **stable hostname** (e.g. `web-0.web`, `web-1.web`). A Deployment +
ClusterIP Service can't give that — you need a **StatefulSet** plus a **headless
Service** (`clusterIP: None`) to publish per-pod DNS records.

## Your task

In namespace `kubelings`:

1. Create a **headless Service** named `web` (`clusterIP: None`) selecting `app=web`.
2. Create a **StatefulSet** named `web` with **≥ 2 replicas**, `serviceName: web`,
   image `ghcr.io/iximiuz/labs/nginx:alpine`.
3. All replicas must be Ready (`web-0`, `web-1`, …).

```sh
kubectl -n kubelings get sts,svc,pods
kubectl -n kubelings exec web-0 -- nslookup web-1.web 2>/dev/null || true
```

<details>
<summary>Hint</summary>

The Service must come first and be headless; the StatefulSet's `serviceName`
must point at it. See `solution.md` for a complete manifest.

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


## Approach

Stable identity = **StatefulSet** + **headless Service**. The headless Service
(`clusterIP: None`) makes the DNS plane publish `web-N.web.<ns>.svc` records; the
StatefulSet's `serviceName` binds the two.

## Create it

```sh
kubectl -n kubelings apply -f - <<'EOF'
apiVersion: v1
kind: Service
metadata:
  name: web
spec:
  clusterIP: None        # headless
  selector: {app: web}
  ports:
    - port: 80
      targetPort: 80
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  serviceName: web        # must match the headless Service
  replicas: 2
  selector:
    matchLabels: {app: web}
  template:
    metadata:
      labels: {app: web}
    spec:
      containers:
        - name: web
          image: ghcr.io/iximiuz/labs/nginx:alpine
          ports: [{containerPort: 80}]
          readinessProbe:
            httpGet: {path: /, port: 80}
            initialDelaySeconds: 1
            periodSeconds: 2
EOF
```

## Verify

```sh
kubectl -n kubelings rollout status sts/web
kubectl -n kubelings get pods -l app=web        # web-0, web-1
kubectl -n kubelings exec web-0 -- nslookup web-1.web
```

</details>
