# Solution — StatefulSet + headless Service

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
